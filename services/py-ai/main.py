from fastapi import FastAPI

app = FastAPI()

@app.get("/health")
def get_python_service_health_status():
    return {"status" : "ok", "service" : "pi-ai"}