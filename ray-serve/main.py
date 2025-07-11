import os
import sys
import time
import base64
import requests

import ray
from ray import serve

# add local modules to path
sys.path.append("../external/tslib")
sys.path.append("../modelling/lib")

from core.image_generator import ImageGenerator
from core.image_service import entrypoint
from core.controller import controllerEntrypoint
from core.autoscaler import autoscalerEndpoint

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
        os.getenv("RAY_SERVE_URL", "http://127.0.0.1:8000") + "/generate",
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
    else:
        print("❌ Request failed:", response.status_code, response.text)

def ray_serve_run():
    import forecast
    import exp
    import data_provider
    import utils
    import models
    import layers
    ray.init(address=os.getenv("RAY_CLIENT_URL", None), 
                runtime_env={
                "working_dir": "./",
                "py_modules": [
                    forecast,
                    exp,
                    data_provider,
                    utils,
                    models,
                    layers
                ]
            })
    serve.start(
        proxy_location ="HeadOnly",
        http_options={
            "host": "0.0.0.0",
            "port": 8000         
        }
    )
    
    serve.run(entrypoint, name="text2img")
    serve.run(controllerEntrypoint, name="controller", route_prefix="/controller")
    serve.run(autoscalerEndpoint, name="autoscaler", route_prefix="/autoscaler")
    
    if "INIT_REPLICAS" in os.environ:
        replicas = int(os.getenv("INIT_REPLICAS", 1))
        print("Setting replicas to", replicas)
        response = requests.post(
            os.getenv("RAY_SERVE_URL", "http://127.0.0.1:8000") + "/controller/replicas",
            params={
                "count": replicas,
            }
        )

        if response.status_code == 200:
            print("✅ Replicas set successfully.")
        else:
            print("❌ Failed to set replicas:", response.status_code, response.text)

    while True:
        time.sleep(3000)

def ray_serve_delete():
    if "RAY_CLIENT_URL" in os.environ:
        ray.init(address=os.getenv("RAY_CLIENT_URL", "auto"))
    else:
        ray.init()
    serve.delete("text2img")
    serve.delete("controller")
    serve.delete("autoscaler")

if __name__ == "__main__": 
    # set MODEL_PATH and other env vars if needed
    if "serve" in sys.argv:
        if "delete" in sys.argv:
            ray_serve_delete()
        else:  
            ray_serve_run()
            if "test" in sys.argv:
                time.sleep(1)
                ray_serve_test()
                ray.shutdown()
    elif "test" in sys.argv:
        run_local_test()
