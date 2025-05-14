import asyncio
import base64
import io
import time
import torch
from fastapi import FastAPI, Query
from ray import serve
from transformers import BertTokenizer, BertForSequenceClassification

# from core.image_generator import ImageGenerator

app = FastAPI()

device = "cuda"

@serve.deployment(
    name="text_service",
    num_replicas=2,
    # autoscaling_config={
    #     "target_ongoing_requests": 2,            # 每个副本理想的并发数
    #     "min_replicas": 1,
    #     "max_replicas": 1,
    #     "upscale_delay_s": 10,                  # 扩容延迟
    #     "downscale_delay_s": 10,                 # 缩容延迟
    #     "metrics_interval_s": 2                 # 采样间隔
    # },
    ray_actor_options={
        "num_gpus": 1,
    },
)
@serve.ingress(app)
class TextService:
    def __init__(self):
        self.lock = asyncio.Lock()
        self.active_requests = 0
        self.tokenizer = BertTokenizer.from_pretrained("bert-base-uncased")
        self.model = BertForSequenceClassification.from_pretrained("bert-base-uncased", num_labels=2)
        self.model.to(device)
        self.model.eval()

    @app.get("/analyze")
    async def analyze(
        self,
        text: str = Query(...),
    ):
        
        self.active_requests += 1
        print(f"[{id}] 🚀 Entering. Active requests: {self.active_requests}")
        try:
            async with self.lock:
                start_time = time.time()
                with torch.no_grad():
                    inputs = self.tokenizer(text, return_tensors="pt", truncation=True, padding=True)
                    inputs = {k: v.to(device) for k, v in inputs.items()}
                    outputs = self.model(**inputs)
                    prediction = outputs.logits.argmax(dim=1).item()
                result = "Positive" if prediction == 1 else "Negative"
                duration = time.time() - start_time

                print(f"'{text}' analysis completed in {duration:.4f} seconds.")

                return {
                    "result": result,
                    "duration": duration,
                }
        finally:
            self.active_requests -= 1
            print(f"[{id}] 🚀 Exiting. Active requests: {self.active_requests}")
    
textEntrypoint = TextService.bind()
