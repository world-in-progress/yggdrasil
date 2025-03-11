from fastapi import FastAPI
from pydantic import BaseModel

class NodeCreateInfo(BaseModel):
    name: str

app = FastAPI()

@app.post("/api/v0/node")
async def create_node(nodeInfo: NodeCreateInfo):
    print(nodeInfo.name)
    return {
        "_id": "YOOOO-The-FIRST-NODE-OF-The-Tree"
    }