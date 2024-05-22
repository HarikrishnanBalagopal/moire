package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	IMG_FIXED_WIDTH  = 480
	IMG_FIXED_HEIGHT = IMG_FIXED_WIDTH
	BW_THRESHOLD     = 255 * 2
)

func bitWiseInvert(xs, ys []byte) {
	xslen := len(xs)
	for i := 0; i < xslen; i++ {
		ys[i] = ^xs[i]
	}
}

func bitWiseAnd(xs, ys, zs []byte) {
	xslen := len(xs)
	for i := 0; i < xslen; i++ {
		zs[i] = xs[i] & ys[i]
	}
}

func bitWiseOr(xs, ys, zs []byte) {
	xslen := len(xs)
	for i := 0; i < xslen; i++ {
		zs[i] = xs[i] | ys[i]
	}
}

func getGifDimensions(gif *gif.GIF) (x, y int) {
	var lowestX int
	var lowestY int
	var highestX int
	var highestY int

	for _, img := range gif.Image {
		if img.Rect.Min.X < lowestX {
			lowestX = img.Rect.Min.X
		}
		if img.Rect.Min.Y < lowestY {
			lowestY = img.Rect.Min.Y
		}
		if img.Rect.Max.X > highestX {
			highestX = img.Rect.Max.X
		}
		if img.Rect.Max.Y > highestY {
			highestY = img.Rect.Max.Y
		}
	}

	return highestX - lowestX, highestY - lowestY
}

func getImageDimensions(rect image.Rectangle) (x, y int) {
	return rect.Max.X - rect.Min.X, rect.Max.Y - rect.Min.Y
}

func gifToFrames(inputPath, outputDir string) ([]*image.NRGBA, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the gif at path '%s' . Error: %w", inputPath, err)
	}
	defer f.Close()
	mygif, err := gif.DecodeAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the gif at path '%s' . Error: %w", inputPath, err)
	}
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make the output directory at path '%s' . Error: %w", outputDir, err)
	}
	// https://stackoverflow.com/questions/33295023/how-to-split-gif-into-images
	imgWidth, imgHeight := getGifDimensions(mygif)
	overpaintImage := image.NewNRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(overpaintImage, overpaintImage.Bounds(), mygif.Image[0], image.Point{}, draw.Src)
	frames := []*image.NRGBA{}
	for i, img := range mygif.Image {
		draw.Draw(overpaintImage, overpaintImage.Bounds(), img, image.Point{}, draw.Over)
		outputFilename := fmt.Sprintf("frame-%d.png", i)
		outputPath := path.Join(outputDir, outputFilename)
		f, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create the output png at path '%s' . Error: %w", outputPath, err)
		}
		if err := png.Encode(f, overpaintImage); err != nil {
			return nil, fmt.Errorf("failed to write the output png at path '%s' . Error: %w", outputPath, err)
		}
		f.Close()
		frame := image.NewNRGBA(image.Rect(0, 0, imgWidth, imgHeight))
		draw.Draw(frame, frame.Bounds(), overpaintImage, image.Point{}, draw.Over)
		frames = append(frames, frame)
	}
	return frames, nil
}

func myResize(srcImg *image.NRGBA, dstW, dstH int) *image.NRGBA {
	// srcRect := srcImg.Bounds()
	// srcW, srcH := getImageDimensions(srcRect)
	dstImg := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcX, srcY := x, y
			srcC := srcImg.NRGBAAt(srcX, srcY)
			dstImg.SetNRGBA(x, y, srcC)
		}
	}
	return dstImg
}

func imgToBlackAndWhite(img *image.NRGBA, threshold uint32, flip bool) *image.NRGBA {
	rect := img.Bounds()
	imgW, imgH := getImageDimensions(rect)
	bwImg := image.NewNRGBA(rect)
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			_srcC := img.At(x, y)
			srcC := _srcC.(color.NRGBA)
			brightness := uint32(srcC.R) + uint32(srcC.G) + uint32(srcC.B)
			var bw uint8 = 0
			// logrus.Infof("r %d g %d b %d a %d brightness %d threshold %d", r, g, b, a, brightness, threshold)
			// logrus.Infof("r %d g %d b %d a %d brightness %d threshold %d", srcC.R, srcC.G, srcC.B, srcC.A, brightness, threshold)
			if flip {
				if brightness <= threshold {
					bw = 255
				}
			} else {
				if brightness > threshold {
					bw = 255
				}
			}
			bwC := color.NRGBA{R: bw, G: bw, B: bw, A: 255}
			bwImg.Set(x, y, bwC)
		}
	}
	return bwImg
}

func normalizeFrames(frames []*image.NRGBA, outputDir string, flipBW bool) ([]*image.NRGBA, error) {
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make the normalized output directory at path '%s' . Error: %w", outputDir, err)
	}
	newFrames := []*image.NRGBA{}
	for i, frame := range frames {
		// fName := dirEntry.Name()
		// var idx int
		// if _, err := fmt.Sscanf(fName, "frame-%d.png", &idx); err != nil {
		// 	return fmt.Errorf("failed to get frame number from the filename. Error: %w", err)
		// }
		// fPath := path.Join(inputDir, fName)
		// logrus.Infof("name: '%s' fPath: '%s' idx: %d", fName, fPath, idx)
		// f, err := os.Open(fPath)
		// if err != nil {
		// 	return fmt.Errorf("failed to open the frame png at path '%s' . Error: %w", fPath, err)
		// }
		// img, err := png.Decode(f)
		// if err != nil {
		// 	return fmt.Errorf("failed to decode the frame png at path '%s' . Error: %w", fPath, err)
		// }
		// {
		// 	logrus.Infof("frame %T", frame)
		// 	testC := frame.At(0, 0)
		// 	logrus.Infof("testC %T %+v", testC, testC)
		// 	r, g, b, a := testC.RGBA()
		// 	logrus.Infof("r %d g %d b %d a %d", r, g, b, a)
		// 	newC := testC.(color.NRGBA)
		// 	logrus.Infof("newC %+v", newC)
		// }
		// _resizedImage := resize.Resize(IMG_FIXED_WIDTH, IMG_FIXED_HEIGHT, frame, resize.Lanczos3)
		// resizedImage := _resizedImage.(*image.NRGBA)
		resizedImage := myResize(frame, IMG_FIXED_WIDTH, IMG_FIXED_HEIGHT)
		bwImg := imgToBlackAndWhite(resizedImage, BW_THRESHOLD, flipBW)
		// if err != nil {
		// 	return fmt.Errorf("failed to convert the resized frame to black and white image. Error: %w", err)
		// }
		outputFilename := fmt.Sprintf("frame-%d.png", i)
		outputPath := path.Join(outputDir, outputFilename)
		df, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create the resized output png at path '%s' . Error: %w", outputPath, err)
		}
		if err := png.Encode(df, bwImg); err != nil {
			return nil, fmt.Errorf("failed to write the resized output png at path '%s' . Error: %w", outputPath, err)
		}
		df.Close()
		newFrames = append(newFrames, bwImg)
		// f.Close()
	}
	// https://stackoverflow.com/questions/22940724/go-resizing-images
	return newFrames, nil
}

func calcMoireFrame(srcFrame *image.NRGBA, mask *image.NRGBA) *image.NRGBA {
	srcRect := srcFrame.Bounds()
	srcW, srcH := getImageDimensions(srcRect)
	dstFrame := image.NewNRGBA(srcRect)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			srcC := srcFrame.NRGBAAt(x, y)
			maskC := mask.NRGBAAt(x, y)
			// black is true and white is false
			var bwC uint8 = 255              // false
			if srcC.R == 0 && maskC.R == 0 { // true and true
				bwC = 0 // true
			}
			c := color.NRGBA{
				R: bwC,
				G: bwC,
				B: bwC,
				A: 255,
			}
			dstFrame.SetNRGBA(x, y, c)
		}
	}
	return dstFrame
}

func calcNewFinalFrame(oldFinalFrame *image.NRGBA, srcFrame *image.NRGBA) *image.NRGBA {
	srcRect := srcFrame.Bounds()
	srcW, srcH := getImageDimensions(srcRect)
	dstFrame := image.NewNRGBA(srcRect)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			oldFinalFrameC := oldFinalFrame.NRGBAAt(x, y)
			srcC := srcFrame.NRGBAAt(x, y)
			// black is true and white is false
			var bwC uint8 = 255                       // false
			if oldFinalFrameC.R == 0 || srcC.R == 0 { // true or true
				bwC = 0 // true
			}
			c := color.NRGBA{
				R: bwC,
				G: bwC,
				B: bwC,
				A: 255,
			}
			dstFrame.SetNRGBA(x, y, c)
		}
	}
	return dstFrame
}

func fillWhiteFalse(src *image.NRGBA) {
	srcRect := src.Bounds()
	srcW, srcH := getImageDimensions(srcRect)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			c := color.NRGBA{
				R: 255,
				G: 255,
				B: 255,
				A: 255,
			}
			src.SetNRGBA(x, y, c)
		}
	}
}

func convertWhiteTransparent(src *image.NRGBA) {
	srcRect := src.Bounds()
	srcW, srcH := getImageDimensions(srcRect)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			c := src.NRGBAAt(x, y)
			if c.R == 255 && c.G == 255 && c.B == 255 { // white
				newC := color.NRGBA{
					R: 0,
					G: 0,
					B: 0,
					A: 0,
				}
				src.SetNRGBA(x, y, newC)
			} else if c.R == 0 && c.G == 0 && c.B == 0 { // black
				// logrus.Infof("black")
			} else {
				logrus.Infof("c %v", c)
			}
		}
	}
}

func calcEntireMoireBackImage(animFrames []*image.NRGBA, maskFrames []*image.NRGBA, outputPath string) (*image.NRGBA, error) {
	moireBackImage := image.NewNRGBA(animFrames[0].Bounds())
	fillWhiteFalse(moireBackImage)
	for i, animFrame := range animFrames {
		maskFrame := maskFrames[i]
		moireFrame := calcMoireFrame(animFrame, maskFrame)
		moireBackImage = calcNewFinalFrame(moireBackImage, moireFrame)
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	if err := png.Encode(f, moireBackImage); err != nil {
		return nil, err
	}
	return moireBackImage, nil
}

func calcEntireMoireFrontImage(mask *image.NRGBA, outputPath string) (*image.NRGBA, error) {
	moireFrontImage := image.NewNRGBA(mask.Bounds())
	draw.Draw(moireFrontImage, moireFrontImage.Bounds(), mask, image.Point{}, draw.Over)
	convertWhiteTransparent(moireFrontImage)
	f, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	if err := png.Encode(f, moireFrontImage); err != nil {
		return nil, err
	}
	return moireFrontImage, nil
}

func main() {
	logrus.Info("main start")
	inputDir := "input"
	outputDir := "output"

	var animFrames []*image.NRGBA
	{
		// animation
		inputAnimationFilename := "balls-3-bounce.gif"
		outputAnimationFramesDir := strings.TrimSuffix(inputAnimationFilename, ".gif")
		animationPath := filepath.Join(inputDir, "animation", inputAnimationFilename)
		animationFramesDir := filepath.Join(outputDir, "animation", outputAnimationFramesDir)
		frames, err := gifToFrames(animationPath, animationFramesDir)
		if err != nil {
			logrus.Fatalf(
				"failed to convert the gif at path '%s' to separate frames at path '%s' . Error: %q",
				animationPath, animationFramesDir, err,
			)
		}
		// normalize
		animationFramesNormalizedDir := filepath.Join(outputDir, "animation-normalized", outputAnimationFramesDir)
		newFrames, err := normalizeFrames(frames, animationFramesNormalizedDir, true)
		if err != nil {
			logrus.Fatalf("failed to normalize the frames. Error: %q", err)
		}
		logrus.Infof("len newFrames %d", len(newFrames))
		animFrames = newFrames
	}

	{
		// mask
		inputAnimationFilename := "vertical-stripes.gif"
		outputAnimationFramesDir := strings.TrimSuffix(inputAnimationFilename, ".gif")
		animationPath := filepath.Join(inputDir, "mask", inputAnimationFilename)
		animationFramesDir := filepath.Join(outputDir, "mask", outputAnimationFramesDir)
		frames, err := gifToFrames(animationPath, animationFramesDir)
		if err != nil {
			logrus.Fatalf(
				"failed to convert the gif at path '%s' to separate frames at path '%s' . Error: %q",
				animationPath, animationFramesDir, err,
			)
		}
		// normalize
		animationFramesNormalizedDir := filepath.Join(outputDir, "mask-normalized", outputAnimationFramesDir)
		newFrames, err := normalizeFrames(frames, animationFramesNormalizedDir, false)
		if err != nil {
			logrus.Fatalf("failed to normalize the frames. Error: %q", err)
		}
		logrus.Infof("len newFrames %d", len(newFrames))
		// normalize flipped masks
		animationFramesNormalizedFlippedDir := filepath.Join(outputDir, "mask-normalized-flipped", outputAnimationFramesDir)
		newFlippedFrames, err := normalizeFrames(frames, animationFramesNormalizedFlippedDir, true)
		if err != nil {
			logrus.Fatalf("failed to normalize the frames. Error: %q", err)
		}
		logrus.Infof("len newFlippedFrames %d", len(newFlippedFrames))
		moireBackImageOutputPath := filepath.Join(outputDir, "moire-back.png")
		if _, err := calcEntireMoireBackImage(animFrames, newFlippedFrames, moireBackImageOutputPath); err != nil {
			logrus.Fatalf("failed to calculate the moire back image. Error: %q", err)
		}
		moireFrontImageOutputPath := filepath.Join(outputDir, "moire-front.png")
		if _, err := calcEntireMoireFrontImage(newFrames[0], moireFrontImageOutputPath); err != nil {
			logrus.Fatalf("failed to calculate the moire front image. Error: %q", err)
		}
	}

	logrus.Info("main end")
}
