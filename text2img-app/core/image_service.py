import base64
import io
from fastapi import FastAPI, Query
from ray import serve

from core.image_generator import ImageGenerator

app = FastAPI()

@serve.deployment(
    num_replicas="auto",
    ray_actor_options={
        "num_gpus": 1,
    },
)
@serve.ingress(app)
class ImageService:
    def __init__(self):
        self.generator = ImageGenerator()

    @app.get("/generate")
    async def generate(
        self,
        index: int = Query(0),
        prompt: str = Query(...),
        steps: int = Query(30),
        scale: float = Query(7),
        sampler_index: str = Query("DPM++ 2M Karras"),
        width: int = Query(512),
        height: int = Query(512)
    ):
        image, duration = self.generator.generate_image(prompt, steps, scale, sampler_index, width, height)

        print(f"Image {index} generation completed in {duration:.4f} seconds.")

        # 转 base64
        buffer = io.BytesIO()
        image.save(buffer, format="PNG")
        encoded_image = base64.b64encode(buffer.getvalue()).decode("utf-8")

        return {
            "image": encoded_image,
            "duration": round(duration, 4)
        }
    
entrypoint = ImageService.bind()
