import time
from fastapi import FastAPI

app = FastAPI()


@app.get("/status/{status_code}")
def status(status_code: int):
    return status_code


@app.get("/response_time/{tt}")
def response_time(tt: int):
    time.sleep(tt)
    return tt


@app.get("/echo_path/{path:path}")
def echo_path(path: str):
    return path
