from fastapi import FastAPI, Query, HTTPException
from ray import serve
import ray
import httpx
import os

from core.image_service import ImageService

RAY_DASHBOARD_URL = os.getenv("RAY_DASHBOARD_URL", "http://localhost:8265")

app = FastAPI()

@serve.deployment(
    name="controller_service",
    num_replicas=1,
    ray_actor_options={
        "num_cpus": 0,
    }
)
@serve.ingress(app)
class ControllerService:
    @app.get("/replicas")
    async def get_replicas(self):
        status = serve.status()
        application = status.applications["text_service"]
        if application:
            deployment = application.deployments["text_service"]
            if deployment:
                running_replicas = deployment.replica_states["RUNNING"]
                all_replicas = sum(deployment.replica_states.values())
                return {
                    "running": running_replicas,
                    "total": all_replicas,
                }
        return {"running": 0, "total": 0}

    @app.post("/replicas")
    async def set_replicas(self, count: int = Query(..., description="Target number of replicas")):
        async with httpx.AsyncClient() as client:
            try:
                response = await client.get(f"{RAY_DASHBOARD_URL}/api/serve/applications/")
                response.raise_for_status()
            except httpx.HTTPError as e:
                raise HTTPException(status_code=500, detail="Unable to retrieve application configurations.")

            data = response.json()
            applications = data.get("applications", {})
    
            application = next((app for name, app in applications.items() if name == "text_service"), None)
            if not application:
                raise HTTPException(status_code=404, detail="Application 'text_service' not found.")
    
            deployments = application.get("deployments", {})
            deployment = next((d for name, d in deployments.items() if name == "text_service"), None)
            deployment = deployment.get("deployment_config", {})
            if not deployment:
                raise HTTPException(status_code=404, detail="Deployment 'text_service' not found.")
    
            if "autoscaling_config" in deployment:
                raise HTTPException(status_code=400, detail="Cannot set 'num_replicas' when 'autoscaling_config' is present.")
    
            deployment["num_replicas"] = count
    
            try:
                put_response = await client.put(
                    f"{RAY_DASHBOARD_URL}/api/serve/applications/",
                    json={"applications": [{
                        "name": "text_service",
                        "import_path": "core.text_service:entrypoint",
                        "deployments": [
                            deployment
                        ],
                    }]},
                )
                put_response.raise_for_status()
            except httpx.HTTPError as e:
                raise HTTPException(status_code=500, detail="Failed to update application configuration.")

        return {"result": "ok"}
    
    @app.get("/status")
    async def get_status(self):
        status = serve.status()
        application = status.applications["text_service"]
        if application:
            deployment = application.deployments["text_service"]
            if deployment:
                return { "status": deployment.status}
        return {"status": "UNKNOWN"}
    

controllerEntrypoint = ControllerService.bind()
