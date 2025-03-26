package imgconv

import (
	"bytes"
	"fmt"
	"github.com/nfnt/resize"
	"image/gif"
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

func Media2Icon(path string, file io.Reader) ([]byte, error) {
	var img image.Image
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
		break
	case ".png":
		img, err = png.Decode(file)
		break
	case ".webp":
		img, err = webp.Decode(file)
		break
	case ".gif":
		img, err = gif.Decode(file)
		break
	default:
		err = fmt.Errorf("unsupported image format: %s", filepath.Ext(path))
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
