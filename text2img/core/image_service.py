import asyncio
import base64
import io
from fastapi import FastAPI, Query
from ray import serve

from core.image_generator import ImageGenerator

app = FastAPI()

@serve.deployment(
    ray_actor_options={
        "num_gpus": 1,
    },
)
@serve.ingress(app)
class ImageService:
    def __init__(self):
        # self.generator = ImageGenerator()
        self.lock = asyncio.Lock()  # 创建一个锁，确保任务串行执行
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
        
        self.active_requests += 1
        print(f"[{id}] 🚀 Entering. Active requests: {self.active_requests}")
        try:
            async with self.lock:
                await asyncio.sleep(1)  # Simulate some processing time
                # image, duration = self.generator.generate_image(prompt, steps, cfg_scale, sampler_index, width, height)

                # print(f"Image {id} generation completed in {duration:.4f} seconds.")

                # # 转 base64
                # buffer = io.BytesIO()
                # image.save(buffer, format="PNG")
                # encoded_image = base64.b64encode(buffer.getvalue()).decode("utf-8")

                # return {
                #     "image": encoded_image,
                #     "duration": round(duration, 4),
                }
        finally:
            self.active_requests -= 1
            print(f"[{id}] 🚀 Exiting. Active requests: {self.active_requests}")
    
entrypoint = ImageService.bind()
