A little utility that randomly selects clips a large corpus of audio and assembles them into a clip.

To run:

```
go get github.com/krig/go-sox
go run *.go -length <chunk length in secs> -count <number of chunks> -dir <folder with audio> -out <output wav>
```
