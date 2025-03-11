from fastapi import FastAPI
from pydantic import BaseModel

class AddInfo(BaseModel):
    a: float
    b: float

app = FastAPI()

@app.post("/api/v0/add")
async def adding(nodeInfo: AddInfo):
    return {
        "result": nodeInfo.a + nodeInfo.b
    }