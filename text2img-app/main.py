import sys
import time
import base64
import requests

from ray import serve
from dotenv import load_dotenv

from image_generator import ImageGenerator
from image_service import entrypoint


def run_local_test():

    generator = ImageGenerator()

    prompt = "A futuristic city with neon lights at night, cyberpunk style"
    steps = 25
    scale = 7.5
    sampler_index = "DPM++ 2M Karras"
    width = 512
    height = 512

    image, duration = generator.generate_image(
        prompt=prompt,
        steps=steps,
        scale=scale,
        sampler_index=sampler_index,
        width=width,
        height=height
    )

    image.save("output/generated_image.png")
    print(f"✅ Image saved as 'generated_image.png' in {duration:.4f} seconds.")

def ray_serve_test():
    response = requests.get(
        "http://127.0.0.1:8000/generate",
        params={
            "id": 0,
            "prompt": "A futuristic city with neon lights at night, cyberpunk style",
            "steps": 20,
            "cfg_scale": 7,
            "sampler_index": "DPM++ 2M Karras",
            "width": 512,
            "height": 512
        }
    )

    if response.status_code == 200:
        data = response.json()
        print(f"✅ Generation took {data['duration']:.4f} seconds.")

        image_data = base64.b64decode(data["image"])
        with open("output/generated_image.png", "wb") as f:
            f.write(image_data)
        print("✅ Test image saved as 'generated_image.png'")
    else:
        print("❌ Request failed:", response.status_code, response.text)


def run_ray_serve():
    serve.run(entrypoint, route_prefix="/")


if __name__ == "__main__": 
    # set MODEL_PATH and other env vars if needed
    if "serve" in sys.argv:
        run_ray_serve()
        if "test" in sys.argv:
            time.sleep(1)
            ray_serve_test()
        while True:
            time.sleep(3600)
    elif "test" in sys.argv:
        run_local_test()
