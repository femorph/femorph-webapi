"""
Example script to interface with the FEMORPH web api at https://api.femorph.com

You'll need to create a .env file containing your FEMORPH_USERNAME and FEMORPH_PASSWORD

"""

import asyncio
from datetime import datetime
import json
import logging
import os
from typing import Any, Dict, Tuple
import uuid

import dotenv
import requests
import websockets

dotenv.load_dotenv()

logging.basicConfig(level=logging.INFO)

USERNAME = os.environ["FEMORPH_USERNAME"]
PASSWORD = os.environ["FEMORPH_PASSWORD"]
CHUNKSIZE = 8192
ADDRESS = "api.femorph.com"
BASE_URL = f"https://{ADDRESS}"
PATH = os.path.dirname(os.path.abspath(__file__))
FEM_PATH = os.path.join(PATH, "data", "cube.cdb")
SURFACE_PATH = os.path.join(PATH, "data", "sphere.ply")

BASE_WS_URL = f"ws://{ADDRESS}/ws/subscribe"


def parse_task_update(data: Dict[str, Any]):
    return {
        "task_id": uuid.UUID(data["task_id"]),
        "status": data["status"],
        "updatedAt": datetime.fromisoformat(data["updatedAt"]),
        "result": data.get("result"),
        "error": data.get("error"),
    }


def assert_healthy() -> None:
    response = requests.get(f"{BASE_URL}/health")
    assert response.status_code == 200 and response.json().get("status") == "ok"
    print("Application Healthy")


async def wait_for_task_completion(task_id: str, user_id: str) -> None:
    url = f"{BASE_WS_URL}/{user_id}"
    async with websockets.connect(url) as ws:
        logging.info("Connected to WebSocket, waiting for task updates...")
        while True:
            try:
                message = await ws.recv()
                data = json.loads(message)
                if data.get("type") == "task_update":
                    try:
                        task_data = parse_task_update(data["payload"])
                    except Exception as e:
                        logging.error("Invalid task update format: %s", str(e))
                        continue
                    if str(task_data["task_id"]) == task_id:
                        logging.info("Task %s status: %s", task_id, task_data["status"])
                        if task_data["status"] == "completed":
                            return
                        elif task_data["status"] == "failed":
                            logging.error(
                                "Task %s failed: %s",
                                task_id,
                                task_data["error"] or "Unknown error",
                            )
                            raise RuntimeError(
                                f"Task {task_id} failed: {task_data['error']}"
                            )
            except websockets.exceptions.ConnectionClosed:
                logging.error("WebSocket connection lost, reconnecting...")
                await asyncio.sleep(1)
                return await wait_for_task_completion(task_id, user_id)


def upload_file(file_path: str, endpoint: str, user_id: str, access_token: str) -> str:
    url = f"{BASE_URL}/users/{user_id}/{endpoint}"
    headers = {
        "Filename": os.path.basename(file_path),
        "Authorization": f"Bearer {access_token}",
    }
    with open(file_path, "rb") as f:
        files = {"file": (os.path.basename(file_path), f, "application/octet-stream")}
        response = requests.post(url, files=files, headers=headers)
    response.raise_for_status()
    logging.info("Upload successful: %s", response.json())
    return response.json()["id"]


def morph_fem(fem_id: str, surface_id: str, user_id: str, access_token: str):
    url = f"{BASE_URL}/users/{user_id}/fems/{fem_id}/morph"
    payload = {"target": surface_id}
    headers = {"Authorization": f"Bearer {access_token}"}
    response = requests.post(url, json=payload, headers=headers)
    response.raise_for_status()
    logging.info("Morph request submitted: %s", response.json())
    task_id = response.json()["task_id"]

    asyncio.run(wait_for_task_completion(task_id, user_id))


def download_nblock(fem_id: str, out_file: str, user_id: str, access_token: str):
    url = f"{BASE_URL}/users/{user_id}/fems/{fem_id}/nblock"
    headers = {"Authorization": f"Bearer {access_token}"}
    response = requests.get(url, stream=True, headers=headers)
    response.raise_for_status()
    with open(out_file, "wb") as f:
        for chunk in response.iter_content(CHUNKSIZE):
            f.write(chunk)
    logging.info("Downloaded FEM nblock: %s", out_file)


def create_user(email: str, password: str, access_token: str) -> None:
    url = f"{BASE_URL}/users/create"
    payload = {"email": email, "password": password}
    headers = {"Authorization": f"Bearer {access_token}"}
    response = requests.post(url, json=payload, headers=headers)
    response.raise_for_status()
    logging.info("User created: %s", email)


def authenticate(username: str, password: str) -> Tuple[str, str]:
    """Authenticate to the femorph-webapi."""
    payload = {"email": username, "password": password}
    response = requests.post(f"{BASE_URL}/auth", json=payload)
    response.raise_for_status()
    dat = response.json()

    return dat["user_id"], dat["access_token"]


if __name__ == "__main__":
    assert_healthy()

    user_id, access_token = authenticate(USERNAME, PASSWORD)
    fem_id = upload_file(FEM_PATH, "fems", user_id, access_token)
    surface_id = upload_file(SURFACE_PATH, "surfaces", user_id, access_token)
    morph_fem(fem_id, surface_id, user_id, access_token)

    out_file = "/tmp/output.inp"
    download_nblock(fem_id, out_file, user_id, access_token)

    assert os.path.isfile(out_file)
    print("PASS")
