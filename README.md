# wyze-mjpeg-proxy

A service providing a MJPEG stream from an RTSP stream.

This was created to take RTSP streams from the [docker-wyze-bridge](https://github.com/mrlt8/docker-wyze-bridge) project and expose them as MJPEG streams so [Mobileraker](https://github.com/Clon1998/mobileraker) can use them.

I created this because `docker-wyze-bridge` is not able to output any streams that work well with `Mobileraker`, and all other options are needlessly complex and/or deprecated like `ffserver`.

This container uses `ffmpeg` to convert the RTSP stream to an MJPEG stream, and then it buffers the last frame in memory, waiting for clients to connect.  Whenn clients ask for a screenshot, it returns the most recent frame instantly (since it already has is). When a client asks for a video stream, it sends the most recent frame and continues sending frames until the client disconnects.  This is very efficient and requires about 300MB of memory for 2 simultaneous 1080p streams.

