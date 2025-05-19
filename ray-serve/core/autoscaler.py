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
        self.observe_length = int(os.getenv("OBSERVE_LENGTH", 3))
        self.forecast_window = int(os.getenv("FORECAST_WINDOW", 36))
        self.data_scale = float(os.getenv("DATA_SCALE", 1))
        self.max_workers = int(os.getenv("MAX_WORKERS", 4))
        self.min_workers = int(os.getenv("MIN_WORKERS", 1))
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
                scale_mean=forecast_mean, scale_std=forecast_std,
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
        obs = self.extract_observation_window(req.time, req.points)
        input_dict = {Columns.OBS: torch.from_numpy(obs).unsqueeze(0)}
        rl_module_out = self.rl_module.forward_inference(input_dict)
        logits = convert_to_numpy(rl_module_out[Columns.ACTION_DIST_INPUTS])
        # get action with the largest probability
        action = int(np.argmax(logits[0]))

        print(f"request observation length: {len(obs)}, get expected worker count: {action + self.min_workers}")

        return {"count": action + self.min_workers}
    

    def extract_observation_window(self, time: int, data: List[DataPoint]) -> np.ndarray:

        # get forecasted data

        obs = np.zeros((7, self.observe_length), dtype=np.float32)

        forecast_metrics =  data[-self.forecast_window:] if len(data) >= self.forecast_window else [None] * (self.forecast_window - len(data)) + data
        recent_new_request = [0 if not m else m.num_new_tasks for m in forecast_metrics]
        recent_timestamp = [time - (self.forecast_window + i + 1) * self.metrics_window for i in range(self.forecast_window)]
        obs[-1] = self.forecaster.forecast(
            enc_data=recent_new_request,
            enc_stamp=recent_timestamp,
        )[self.observe_length-1::-1]

        recent = data[-self.observe_length:] if len(data) >= self.observe_length else [None] * (self.observe_length - len(data)) + data
        for i, point in enumerate(recent):
            if point is None:
                continue  # Keep default zeros for padding

            features = [
                ("running_workers", point.active_workers),
                ("new_requests", point.num_new_tasks),
                ("ongoing_requests", point.num_ongoing_tasks),
                ("finished_requests", point.num_completed_tasks),
                ("requests_delay", point.avg_delay),
                ("requests_duration", point.avg_duration),
                ("forecasted_requests", obs[i][-1]),
            ]

            for row, (key, raw_value) in enumerate(features):
                min_val = self.FEATURE_MIN[key]
                max_val = self.FEATURE_MAX[key]
                # Min-max normalization with small epsilon for safety
                norm_val = (raw_value - min_val) / (max_val - min_val + 1e-6)
                norm_val = np.clip(norm_val, 0.0, 1.0)  # 防止越界
                obs[row, i] = norm_val

        # flatten the observation to 1D array
        obs = obs.flatten()

        return obs
    
autoscalerEndpoint = AutoscalerService.bind()
