from fastapi import FastAPI, Query, HTTPException
from ray import serve
import ray
import httpx
import os
import numpy as np
from pydantic import BaseModel
from typing import List

from ray.rllib.core import DEFAULT_MODULE_ID
from ray.rllib.core.columns import Columns
from ray.rllib.core.rl_module.rl_module import RLModule
from ray.rllib.utils.numpy import convert_to_numpy

import torch

from forecast.tslib_util import (
    TimeseriesForecaster,
    TimeseriesTransformer
)

app = FastAPI()

class DataPoint(BaseModel):
    active_workers: int
    num_new_tasks: int
    num_ongoing_tasks: int
    num_completed_tasks: int
    avg_duration: float  # in milliseconds
    avg_delay: float     # in milliseconds

class CalcWorkerCountRequestParam(BaseModel):
    time: int
    points: List[DataPoint]

@serve.deployment(
    name="autoscaler_service",
    num_replicas=1,
    ray_actor_options={
        "num_cpus": 0,
    }
)
@serve.ingress(app)
class AutoscalerService:

    def __init__(self):
        self.data_scale = float(os.getenv("DATA_SCALE", 1))
        self.max_workers = int(os.getenv("MAX_WORKERS", 1))
        self.min_workers = int(os.getenv("MIN_WORKERS", 1))

        self.scaler_type = os.getenv("SCALER_TYPE", "threshold")
        if self.scaler_type == "threshold":
            self.scaler_threshold = float(os.getenv("SCLAER_THRESHOLD", 4.0))
        else:
            rl_model_path = os.getenv("RL_MODEL_PATH", "")
            forecast_model_path = os.getenv("FORECAST_MODEL_PATH", "")
            if os.path.exists(forecast_model_path) is False:
                raise ValueError(f"Forecast model path {forecast_model_path} does not exist.")
            if os.path.exists(rl_model_path) is False:
                raise ValueError(f"RL model path {rl_model_path} does not exist.")

            forecast_base_time = int(os.getenv("FORECAST_BASE_TIME", 0))
            forecast_mean = float(os.getenv("FORECAST_MEAN", 0.0))
            forecast_std = float(os.getenv("FORECAST_STD", 1.0))
            self.metrics_window = int(os.getenv("METRICS_WINDOW", 10))
            self.forecast_window = int(os.getenv("FORECAST_WINDOW", 36))
            self.observe_length = int(os.getenv("OBSERVE_LENGTH", 3))
            self.future_length = int(os.getenv("FUTURE_LENGTH", 3))

            self.rl_module = RLModule.from_checkpoint(
                os.path.join(
                    rl_model_path,
                    "learner_group",
                    "learner",
                    "rl_module",
                    DEFAULT_MODULE_ID,
                )
            )
            self.forecaster = TimeseriesForecaster(ckpt_path=forecast_model_path)
            self.forecaster.setTransformer(
                transformer=TimeseriesTransformer(
                    date_start=forecast_base_time, date_scale=120, scale=True,
                    scale_mean=forecast_mean * self.data_scale, scale_std=forecast_std * self.data_scale,
                )
            )
            self.FEATURE_MIN = {
                "running_workers": 0,
                "new_requests": 0,
                "ongoing_requests": 0,
                "finished_requests": 0,
                "requests_delay": 0.0,
                "requests_duration": 0.0,
                "forecasted_requests": 0.0,
            }

            self.FEATURE_MAX = {
                "running_workers": self.max_workers,         
                "new_requests": self.data_scale * 10,            
                "ongoing_requests": self.data_scale * 50,        
                "finished_requests": self.max_workers * 6,       
                "requests_delay": self.data_scale * 20000,          
                "requests_duration": 12000, 
                "forecasted_requests": self.data_scale * 10,      
            }

    @app.post("/calc")
    async def calc(self, req: CalcWorkerCountRequestParam):
        if not req.points:
            raise HTTPException(status_code=400, detail="No data points provided.")
        if self.scaler_type == "threshold":
            ongoing_tasks = req.points[-1].num_ongoing_tasks + req.points[-1].num_new_tasks
            expected_workers = np.ceil(ongoing_tasks / self.scaler_threshold)
            expected_workers = int(max(self.min_workers, min(expected_workers, self.max_workers)))

            print(f"Ongoing tasks: {ongoing_tasks}, get expected worker count: {expected_workers}")
            return {"count": expected_workers}
            
        else:
            obs = self.extract_observation_window(req.time, req.points)
            input_dict = {Columns.OBS: torch.from_numpy(obs).unsqueeze(0)}
            rl_module_out = self.rl_module.forward_inference(input_dict)
            logits = convert_to_numpy(rl_module_out[Columns.ACTION_DIST_INPUTS])
            # get action with the largest probability
            action = int(np.argmax(logits[0]))

            print(f"request observation length: {len(req.points)}, get expected worker count: {action + self.min_workers}")
            return {"count": action + self.min_workers}
    

    def extract_observation_window(self, time: int, data: List[DataPoint]) -> np.ndarray:

        obs = np.zeros((self.observe_length * 6 + self.future_length,), dtype=np.float32)
        if len(data) == 0:
            return obs

        if len(data) >= self.forecast_window:
            forecast_metrics =  data[-self.forecast_window:]
            recent_new_request = [0 if not m else m.num_new_tasks for m in forecast_metrics]
            recent_timestamp = [time - i * self.metrics_window for i in range(self.forecast_window-1, -1, -1)]
            future_requests = self.forecaster.forecast(
                enc_data=recent_new_request,
                enc_stamp=recent_timestamp,
            )[:self.future_length]
            print(recent_new_request)
            print(future_requests)
        else:
            future_requests = np.zeros(self.future_length, dtype=np.float32)

        recent = data[-self.observe_length:] if len(data) >= self.observe_length else [None] * (self.observe_length - len(data)) + data
        for i, point in enumerate(recent + list(future_requests)):
            if point is None:
                continue  # Keep default zeros for padding

            observed = isinstance(point, DataPoint)
            features = [
                ("running_workers", point.active_workers),
                ("requests_delay", point.avg_delay),
                ("requests_duration", point.avg_duration),
                ("ongoing_requests", point.num_ongoing_tasks),
                ("finished_requests", point.num_completed_tasks),
                ("new_requests", point.num_new_tasks)
            ] if observed else [
                ("new_requests", point)
            ]

            for row, (key, raw_value) in enumerate(features):
                min_val = self.FEATURE_MIN[key]
                max_val = self.FEATURE_MAX[key]
                # Min-max normalization with small epsilon for safety
                norm_val = (raw_value - min_val) / (max_val - min_val + 1e-6)
                norm_val = np.clip(norm_val, 0.0, 1.0)
                if i < self.observe_length:
                    obs[row * self.observe_length + i] = norm_val
                else:
                    obs[6 * self.observe_length + i - self.observe_length] = norm_val

        return obs
    
autoscalerEndpoint = AutoscalerService.bind()
