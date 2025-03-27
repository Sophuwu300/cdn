package imgconv

import (
	"bytes"
	"fmt"
	"github.com/nfnt/resize"
	"image/gif"
	"os"
	"os/exec"
	"strings"

	// "gocv.io/x/gocv"
	"golang.org/x/image/webp"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
)

// ffmpeg -i file.mp4 -vf "select='eq(n\,20)+eq(n\,40)+eq(n\,60)+eq(n\,80)'" -vsync 0 frame%d.png

func scaleEncode(img image.Image) ([]byte, error) {
	var w, h uint
	aspect := float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
	if aspect > 1 {
		w = 128
		h = uint(float64(w) / aspect)
	} else {
		h = 128
		w = uint(float64(h) * aspect)
	}
	img = resize.Resize(w, h, img, resize.NearestNeighbor)
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image to PNG: %v", err)
	}
	return buf.Bytes(), nil
}

func getFirstFrame(path string) (image.Image, error) {

	// ffmpeg -i input.mp4 -vframes 1 -f image2pipe -vcodec png -
	// -vf "select='eq(pict_type\,I)'" -frames:v 1
	var out bytes.Buffer
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-i", path, "-vf", "select='eq(pict_type\\,I)'", "-frames:v", "1", "-vsync", "0", "-f", "image2pipe", "-c:v", "png", "-")
	// cmd.Stdin = file
	cmd.Stdout = &out
	cmd.Stderr = os.Stdout
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error running ffmpeg: %v", err)
	}
	var img image.Image
	img, err = png.Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}
	return img, nil
}

var medidaFuncs = map[string]func(file io.Reader) (image.Image, error){
	".jpg":  jpeg.Decode,
	".jpeg": jpeg.Decode,
	".png":  png.Decode,
	".webp": webp.Decode,
	".gif":  gif.Decode,
}

func Media2Icon(path string) ([]byte, error) {
	var img image.Image
	var err error
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".mp4" {
		img, err = getFirstFrame(path)
	} else if fn, ok := medidaFuncs[ext]; ok {
		var file *os.File
		file, err = os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("error opening file: %v", err)
		}
		img, err = fn(file)
		file.Close()
	} else {
		err = fmt.Errorf("unsupported file type: %s", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}
	return scaleEncode(img)
}

/*
func getFirstFrameAsPNGBytes(videoPath string) ([]byte, error) {
	// Open the video file
	video, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		return nil, fmt.Errorf("error opening video: %v", err)
	}
	img := gocv.NewMat()
	ok := video.Read(&img)

	if !ok || img.Empty() {
		img.Close()
		video.Close()
		return nil, fmt.Errorf("failed to read the first frame")
	}

	imgImg, err := img.ToImage()
	img.Close()
	video.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to convert frame to image: %v", err)
	}
	return scaleEncode(imgImg)
}
*/
