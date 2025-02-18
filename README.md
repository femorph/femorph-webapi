## FEMORPH Web API

A public web API for FEMORPH can be accessed at [api.femorph.com](https://api.femorph.com), which contains a variety of endpoints to allow you to upload both FEMs and surfaces for mesh metamorphosis.

API docs are available at [api.femorph.com/docs](https://api.femorph.com/docs).

![Swagger UI](docs/swagger-ui.png)

### Graphical Web Interface

The public API also exposes a graphical user interface which can be accessed from a modern browser by visiting [api.femorph.com](https://api.femorph.com).

![Web GUI](docs/web-gui.png)

Credentials for both this graphical web application and the API can be obtained by visiting [femorph.com](https://www.femorph.com/) and contacting sales.

### Examples

#### Python

The file `example.py` contains everything you need to upload and morph the `cube.cdb` file to the `sphere.ply` file. Once you've obtained your credentials, simply install the requirements and run `python example.py`:

```bash
$ pip install -r requirements.txt
$ python example.py 
Application Healthy
INFO:root:Upload successful: {'filename': 'cube.cdb', 'id': 'c0cf7b82-4735-49e6-9ebb-7358a252bef3', 'dataHash': '0a54ea4bacd3f638c5e069611ba8f18c5dcaeb3867d9067c864f5d8f5cdf9c44', 'dataType': 'FemArtifact', 'modified': False, 'nSectors': None, 'axis': None}
INFO:root:Upload successful: {'filename': 'sphere.ply', 'id': 'cb5c815b-890e-4893-a0dd-f99ad69012a6', 'dataHash': '6b49f02f1ca9cc28cc7507c94198d6c100e8f31d888f8c1660c4ee78e2ab2ce1', 'dataType': 'SurfaceArtifact', 'modified': False, 'nSectors': None, 'axis': None}
INFO:root:Morph request submitted: {'message': 'Submitted morph for fem c0cf7b82-4735-49e6-9ebb-7358a252bef3', 'task_id': '3ed5a1a0-c172-446d-9146-d5ce5393ad31'}
INFO:root:Connected to WebSocket, waiting for task updates...
INFO:root:Task 3ed5a1a0-c172-446d-9146-d5ce5393ad31 status: completed
INFO:root:Downloaded FEM nblock: /tmp/output.inp
PASS
```

Results can be visualized with [mapdl-archive](https://github.com/akaszynski/mapdl-archive).

#### Go

Run `femorph_client.go` to upload and morph the `cube.cdb` file to the `sphere.ply` file and download the resulting nodes. Be sure you've obtained your credentials first and entered them in a `.env` file in the format of:

```
FEMORPH_USERNAME=<USER>
FEMORPH_PASSWORD=<PASS>
```

Running the go program:

```bash
$ go mod init femorph-client
$ go mod tidy
$ go run femorph_client.go 
2025/02/13 06:22:05 Application Healthy
2025/02/13 06:22:06 Received access token for <REDACTED>
2025/02/13 06:22:07 Cleared user session
2025/02/13 06:22:17 Upload successful: c9690ecb-e1e8-8831-456a-67e6059aa416
2025/02/13 06:22:18 Upload successful: 37726432-31fb-9168-4022-210efc759d30
2025/02/13 06:22:19 Morph task d2b62258-8e6e-4ebc-4556-47aeb7e06220 started
2025/02/13 06:22:19 starting...
2025/02/13 06:22:19 Waiting for task d2b62258-8e6e-4ebc-4556-47aeb7e06220...
2025/02/13 06:22:20 Task d2b62258-8e6e-4ebc-4556-47aeb7e06220 status: completed
```

