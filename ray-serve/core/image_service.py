import asyncio
import base64
import io
import os
from fastapi import FastAPI, Query
from ray import serve

from core.image_generator import ImageGenerator

app = FastAPI()

@serve.deployment(
    name="image_service",
    num_replicas=6,  
    # autoscaling_config={
    #     "target_ongoing_requests": 2,           # 每个副本理想的并发数
    #     "min_replicas": 1,
    #     "max_replicas": 3,
    #     "upscale_delay_s": 10,                  # 扩容延迟
    #     "downscale_delay_s": 10,                # 缩容延迟
    #     "metrics_interval_s": 10,                # 采样间隔
    #     "look_back_period_s": 10
    # },
    ray_actor_options={
        "num_gpus": 1,
    },
)
@serve.ingress(app)
class ImageService:
    def __init__(self):
        self.generator = ImageGenerator()
        self.lock = asyncio.Lock()
        self.active_requests = 0

    @app.get("/generate")
    async def generate(
        self,
        id: str = Query("0"),
        prompt: str = Query(...),
        steps: int = Query(30),
        cfg_scale: float = Query(7),
        sampler_index: str = Query("DPM++ 2M Karras"),
        width: int = Query(512),
        height: int = Query(512)
    ):
        try:
            async with self.lock:
                _, duration = self.generator.generate_image(prompt, steps, cfg_scale, sampler_index, width, height)

                print(f"Image {id} generation completed in {duration:.4f} seconds.")

                return {
                    "duration": duration,
                }
        except Exception as e:
            print(f"Error generating image {id}: {e}")
            return {
                "duration": 0,
                "error": str(e),
            }

    
entrypoint = ImageService.bind()
