import os
import time
import torch
from diffusers import StableDiffusionPipeline
from diffusers import (
    DDIMScheduler,
    PNDMScheduler,
    EulerDiscreteScheduler,
    EulerAncestralDiscreteScheduler,
    HeunDiscreteScheduler,
    DPMSolverMultistepScheduler,
    DPMSolverSinglestepScheduler,
    KDPM2DiscreteScheduler,
    KDPM2AncestralDiscreteScheduler,
    LMSDiscreteScheduler,
    DEISMultistepScheduler,
    UniPCMultistepScheduler
)


class ImageGenerator:
    def __init__(self, model_path: str = "", use_xformers: bool = True):
        self.model_path = model_path if model_path else os.getenv("MODEL_PATH")
        self.pipe = None
        self.use_xformers = use_xformers
        self.load_model()

    def load_model(self):
        print(f"Loading model from: {self.model_path}")
        start_time = time.time()
        self.pipe = StableDiffusionPipeline.from_single_file(
            self.model_path,
            torch_dtype=torch.float32,
            use_safetensors=True,
        )
        self.pipe.set_progress_bar_config(disable=True)

        if torch.cuda.is_available():
            self.pipe.to("cuda")

        if self.use_xformers:
            self.pipe.enable_xformers_memory_efficient_attention()

        print(f"Model loaded successfully in {time.time() - start_time:.2f} seconds.")

    def sampler_index_to_scheduler(self, name: str):
        scheduler_map = {
            "DDIM": (DDIMScheduler, {}),
            "PLMS": (PNDMScheduler, {}),
            "Euler": (EulerDiscreteScheduler, {}),
            "Euler a": (EulerAncestralDiscreteScheduler, {}),
            "Heun": (HeunDiscreteScheduler, {}),
            "LMS": (LMSDiscreteScheduler, {}),
            "LMS Karras": (LMSDiscreteScheduler, {"use_karras_sigmas": True}),
            "DPM2": (KDPM2DiscreteScheduler, {}),
            "DPM2 Karras": (KDPM2DiscreteScheduler, {"use_karras_sigmas": True}),
            "DPM2 a": (KDPM2AncestralDiscreteScheduler, {}),
            "DPM2 a Karras": (KDPM2AncestralDiscreteScheduler, {"use_karras_sigmas": True}),
            "DPM++ 2M": (DPMSolverMultistepScheduler, {}),
            "DPM++ 2M Karras": (DPMSolverMultistepScheduler, {"use_karras_sigmas": True}),
            "DPM++ 2M SDE": (DPMSolverMultistepScheduler, {"algorithm_type": "sde-dpmsolver++"}),
            "DPM++ 2M SDE Karras": (DPMSolverMultistepScheduler, {"use_karras_sigmas": True, "algorithm_type": "sde-dpmsolver++"}),
            "DPM++ SDE": (DPMSolverSinglestepScheduler, {}),
            "DPM++ SDE Karras": (DPMSolverSinglestepScheduler, {"use_karras_sigmas": True}),
            "DEIS": (DEISMultistepScheduler, {}),
            "UniPC": (UniPCMultistepScheduler, {}),
        }

        SchedulerClass, kwargs = scheduler_map.get(name, (DPMSolverMultistepScheduler, {}))
        return SchedulerClass.from_config(self.pipe.scheduler.config, **kwargs)

    def generate_image(self, prompt: str, steps: int, scale: float, sampler_index: str, width: int, height: int):
        scheduler = self.sampler_index_to_scheduler(sampler_index)
        self.pipe.scheduler = scheduler

        start_time = time.time()
        result = self.pipe(
            prompt=prompt,
            num_inference_steps=steps,
            guidance_scale=scale,
            width=width,
            height=height,
        )
        duration = time.time() - start_time

        image = result.images[0]
        return image, duration
