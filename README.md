# Nice HTTP Server for file sharing
the repo is called 'cdn' because it's a short name.
This is just a HTTP server made for file sharing.
With some nice features...

## Features
- Modern UI
- Automatic scaling and responsive design with vanilla CSS and less than 40 lines of JS.
- Supports large files and video streaming.
- Automatic thumbnail generation for images and videos.
- Custom-made icon font, 99.65% more space efficient than font-awesome.

- Efficient golang back-end uses only 200 mb of RAM with 10 concurrent unique HD video streams.
- Full streaming is supported, so you can start watching videos before they are fully downloaded.
- When no connections are active, the server uses too little CPU to measure.

#### Pretty Front-end
- only 115 lines of JavaScript, no dependencies.
- No frameworks, no libraries, no dependencies.
- Works on all browsers, with interactive UI.
- Users can sort files by name, size, date, type.
- Automatic thumbnail resizing using back-end image processing.

<img src="https://cdn.sophuwu.com/img/newcdn.png" />


## Installation
### Build with Go
Clone the repository and build it with Go. Make sure you have Go installed on your system.
```bash
git clone git.sophuwu.com/cdn
cd cdn
go build -ldflags="-w -s" -trimpath -o build/
```
The binary will be created in the `build/` directory.
### Run the server
To run the server, you need to set the environment variables for the port, address, HTTP directory, and database path. You can do this in your terminal or create a `.env` file.

#### Variables
- `PORT`: The port on which the server will listen (default: 8080).
- `ADDR`: The address on which the server will listen (default: 127.0.0.1).
- `HTTP_DIR`: The directory to serve.
- `DB_PATH`: The Database to cache generated thumbnails.

#### Example 1
```bash
export PORT=8069
export ADDR=0.0.0.0
export HTTP_DIR=/path/to/your/files
export DB_PATH=/dir/dir/thumbnails.db
./build/cdn
```
#### Example 2
```bash
PORT=8069 ADDR=0.0.0.0 HTTP_DIR=/path/to/your/files DB_PATH=/dir/dir/thumbnails.db ./build/cdn
```

#### Installation after build
```bash
sudo install ./build/cdn /usr/local/bin/cdnd
```

### Systemd Service
Edit the `cdnd.service` file and put it in `/etc/systemd/system/`. Make sure to set the correct paths for `HTTP_DIR` and `DB_PATH`.

If you prefer to write the service file manually, here is an example:
```
[Unit]
Description=File Sharing HTTP Server
After=network.target

[Service]
WorkingDirectory=/home/httpuser/files
ExecStart=/usr/local/bin/cdnd
Type=simple

User=httpuser

Environment=DB_PATH=/home/httpuser/files.db
Environment=HTTP_DIR=/home/httpuser/files
Environment=PORT=9889
Environment=ADDR=127.0.0.1

[Install]
WantedBy=multi-user.target
```

Then run:
```bash
sudo systemctl daemon-reload
sudo systemctl enable cdnd
sudo systemctl start cdnd
```