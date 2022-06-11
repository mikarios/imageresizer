package imagehelper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color" // nolint:misspell // nothing I can do about it
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/services/cdnservice"
	"github.com/mikarios/imageresizer/internal/services/config"
	"github.com/mikarios/imageresizer/pkg/dtos/imagedto"
)

const (
	jpgExtension  = "jpg"
	jpegExtension = "jpeg"
	pngExtension  = "png"
	webpExtension = "webp"
)

var (
	errUnsupportedFileType = errors.New("unsupported file type")
	errNoDimensionsDefined = errors.New("no dimensions defined")
	errUploadingImage      = errors.New("could not upload image to cdn")
	errScalingImage        = errors.New("could not scale image")
	errCropImage           = errors.New("could not crop image")
	errMinXMaxY            = errors.New("could not scale minXmaxY")
	errMinYMaxX            = errors.New("could not scale minYmaxX")
	errDeletingImage       = errors.New("could not delete image")
)

type ImageJob struct {
	*imagedto.ImageStruct
	ShopID         int                     `json:"shopID"`
	ImageExtension string                  `json:"imageExtension"`
	ImagesOnCdn    *map[string]interface{} `json:"-"`
	DeleteImages   []string                `json:"deleteImages"`
}

type ProcessImageError struct {
	URL string `json:"url"`
	Err string `json:"err"`
	Msg string `json:"msg"`
	Dim string `json:"dim"`
}

func (e *ProcessImageError) Error() string {
	return fmt.Sprintf("Err: %v, url: %s, dimensions: %s, msg: %s", e.Err, e.URL, e.Dim, e.Msg)
}

func ProcessJobImage(ctx context.Context, imageJob *ImageJob) []error {
	logger.Debug(ctx, "processing photo for shop", imageJob.ShopID, *imageJob)

	cdn := cdnservice.GetInstance()
	cfg := config.GetInstance()

	collectedErrors := make([]error, 0)

	for i, imagePath := range imageJob.DeleteImages {
		idx := strings.Index(imagePath, cfg.CDN.ImagesFolder)
		if idx > 0 {
			imageJob.DeleteImages[i] = imagePath[idx:]
		}
	}

	if imageJob.ImageStruct != nil { // this is not a deletion job, this needs to process the image
		processScaleImageJob(ctx, cdn, imageJob, cfg, &collectedErrors)
	}

	now := time.Now()

	if err := cdn.Delete(cfg.CDN.Bucket, imageJob.DeleteImages); err != nil {
		errProcessImage := &ProcessImageError{URL: imageJob.URL, Err: errDeletingImage.Error(), Msg: err.Error()}
		collectedErrors = append(collectedErrors, errProcessImage)
	}

	logger.Debug(ctx, fmt.Sprintf("deleting photos finished. Took: %v", time.Since(now)))

	if len(collectedErrors) > 0 {
		return collectedErrors
	}

	return nil
}

func UploadMainProductImageToCDN(
	ctx context.Context,
	cdn *cdnservice.CdnStruct,
	shopID *int,
	productID,
	imageName,
	imagesFolder,
	imgURL string,
	img []byte,
	imagesOnCDN *map[string]interface{}, // nolint:gocritic // this should be pointer
) (downloadedImage []byte, err error) {
	fullImagePath := ImageSubPath("", shopID, productID, nil, nil, nil, nil, imageName)
	fullImagePath = path.Join(imagesFolder, fullImagePath)
	downloadedImage = img

	if imagesOnCDN != nil {
		if _, ok := (*imagesOnCDN)[fullImagePath]; ok {
			return nil, err
		}
	}

	if len(downloadedImage) == 0 {
		downloadedImage, err = downloadImage(ctx, imgURL)
		if err != nil {
			return nil, fmt.Errorf("could not download image [%v]: %w", imgURL, err)
		}
	}

	contentType := http.DetectContentType(downloadedImage)
	if err = cdn.StoreFile("", fullImagePath, bytes.NewReader(downloadedImage), contentType); err != nil {
		err = fmt.Errorf("could not store full image: %w", err)
	}

	return downloadedImage, err
}

// ImageSubPath calculates the correct image path based on the data provided. EITHER scaleDimension OR cropDimensions
// should have value. If both have one will be ignored.
func ImageSubPath(
	prefix string,
	shopID *int,
	productID string,
	scaleDimension *int,
	cropDimensions,
	minXMaxY,
	minYMaxX *imagedto.Dimensions,
	fileName string,
) string {
	shopIDStr := ""

	if shopID != nil {
		shopIDStr = strconv.Itoa(*shopID)
	}

	p := path.Join(prefix, shopIDStr, productID)

	switch {
	case scaleDimension != nil:
		p = path.Join(p, strconv.Itoa(*scaleDimension))
	case cropDimensions != nil:
		p = path.Join(p, fmt.Sprintf("%vx%v", cropDimensions.X, cropDimensions.Y))
	case minXMaxY != nil:
		p = path.Join(p, "minxmaxy", fmt.Sprintf("%vx%v", minXMaxY.X, minXMaxY.Y))
	case minYMaxX != nil:
		p = path.Join(p, "minymaxx", fmt.Sprintf("%vx%v", minYMaxX.X, minYMaxX.Y))
	default:
		p = path.Join(prefix, shopIDStr, productID)
	}

	return path.Join(p, fileName)
}

func processScaleImageJob(
	ctx context.Context,
	cdn *cdnservice.CdnStruct,
	imageJob *ImageJob,
	cfg *config.Config,
	collectedErrors *[]error,
) {
	var img []byte = nil

	now := time.Now()

	img, err := UploadMainProductImageToCDN(
		ctx,
		cdn,
		&imageJob.ShopID,
		imageJob.ProductID,
		imageJob.Name,
		cfg.CDN.ImagesFolder,
		imageJob.URL,
		img,
		imageJob.ImagesOnCdn,
	)
	if err != nil {
		errProcessImage := &ProcessImageError{URL: imageJob.URL, Err: errUploadingImage.Error(), Msg: err.Error()}
		*collectedErrors = append(*collectedErrors, errProcessImage)
	}

	start := time.Now()
	extension := imageJob.ImageExtension

	for _, scaleDimension := range imageJob.ScaleDimensionMax {
		img, err = handleScaleImage(ctx, imageJob, &cfg.CDN, scaleDimension, img, cdn, extension)
		if err != nil {
			errProcessImage := &ProcessImageError{
				URL: imageJob.URL,
				Err: errScalingImage.Error(),
				Msg: err.Error(),
				Dim: fmt.Sprintf("%d", *scaleDimension),
			}
			*collectedErrors = append(*collectedErrors, errProcessImage)
		}
	}

	for _, cropDimension := range imageJob.CropDimensions {
		img, err = handleCropImage(ctx, imageJob, &cfg.CDN, cropDimension, img, cdn, extension)
		if err != nil {
			errProcessImage := &ProcessImageError{
				URL: imageJob.URL,
				Err: errCropImage.Error(),
				Msg: err.Error(),
				Dim: fmt.Sprintf("%dx%d", cropDimension.X, cropDimension.Y),
			}
			*collectedErrors = append(*collectedErrors, errProcessImage)
		}
	}

	for _, minXMaxY := range imageJob.MinXMaxY {
		img, err = handleMinXMaxYImage(ctx, imageJob, &cfg.CDN, minXMaxY, img, cdn, extension)
		if err != nil {
			errProcessImage := &ProcessImageError{
				URL: imageJob.URL,
				Err: errMinXMaxY.Error(),
				Msg: err.Error(),
				Dim: fmt.Sprintf("%dx%d", minXMaxY.X, minXMaxY.Y),
			}
			*collectedErrors = append(*collectedErrors, errProcessImage)
		}
	}

	for _, minYMaxX := range imageJob.MinYMaxX {
		img, err = handleMinYMaxXImage(ctx, imageJob, &cfg.CDN, minYMaxX, img, cdn, extension)
		if err != nil {
			errProcessImage := &ProcessImageError{
				URL: imageJob.URL,
				Err: errMinYMaxX.Error(),
				Msg: err.Error(),
				Dim: fmt.Sprintf("%dx%d", minYMaxX.X, minYMaxX.Y),
			}
			*collectedErrors = append(*collectedErrors, errProcessImage)
		}
	}

	logger.Debug(ctx, fmt.Sprintf("Scaling ALL %v took: %v", imageJob.Name, time.Since(start)))

	logger.Debug(
		ctx,
		fmt.Sprintf("processing photo %v for shop ID: %v finished. Took: %v",
			imageJob.Name, imageJob.ShopID, time.Since(now),
		),
	)
}

func handleScaleImage(
	ctx context.Context,
	imageJob *ImageJob,
	cdnConfig *config.CDNConfig,
	scaleDimension *int,
	img []byte,
	cdn *cdnservice.CdnStruct,
	extension string,
) (downloadedImage []byte, err error) {
	imagePath := ImageSubPath("", &imageJob.ShopID, imageJob.ProductID, scaleDimension, nil, nil, nil, imageJob.Name)
	downloadedImage = img

	if imageJob.ImagesOnCdn != nil {
		if _, ok := (*imageJob.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; ok {
			return downloadedImage, nil
		}
	}

	if len(downloadedImage) == 0 {
		downloadedImage, err = downloadImage(ctx, imageJob.URL)
		if err != nil {
			return nil, fmt.Errorf("could not download image [%v]: %w", imageJob.URL, err)
		}
	}

	output, err := scaleImage(&downloadedImage, scaleDimension, extension)
	if err != nil {
		return downloadedImage, fmt.Errorf("could not scale %v to %v: %w", imagePath, *scaleDimension, err)
	}

	contentType := http.DetectContentType(downloadedImage)
	if err = cdn.StoreFile("", path.Join(cdnConfig.ImagesFolder, imagePath), output, contentType); err != nil {
		return downloadedImage, fmt.Errorf("could not store image %v to cdn: %w", imagePath, err)
	}

	return downloadedImage, nil
}

func handleCropImage(
	ctx context.Context,
	imageJob *ImageJob,
	cdnConfig *config.CDNConfig,
	cropDimension *imagedto.Dimensions,
	img []byte,
	cdn *cdnservice.CdnStruct,
	extension string,
) (downloadedImage []byte, err error) {
	downloadedImage = img
	imagePath := ImageSubPath("", &imageJob.ShopID, imageJob.ProductID, nil, cropDimension, nil, nil, imageJob.Name)

	if imageJob.ImagesOnCdn != nil {
		if _, ok := (*imageJob.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; ok {
			return downloadedImage, nil
		}
	}

	if len(downloadedImage) == 0 {
		downloadedImage, err = downloadImage(ctx, imageJob.URL)
		if err != nil {
			return nil, fmt.Errorf("could not download image [%v]: %w", imageJob.URL, err)
		}
	}

	output, err := cropImage(&downloadedImage, cropDimension, extension)
	if err != nil {
		err = fmt.Errorf("could not scale %v to %vx%v: %w", imagePath, cropDimension.X, cropDimension.Y, err)
		return downloadedImage, err
	}

	contentType := http.DetectContentType(downloadedImage)
	if err = cdn.StoreFile("", path.Join(cdnConfig.ImagesFolder, imagePath), output, contentType); err != nil {
		return downloadedImage, fmt.Errorf("could not store image %v to cdn: %w", imagePath, err)
	}

	return downloadedImage, nil
}

func handleMinXMaxYImage(
	ctx context.Context,
	imageJob *ImageJob,
	cdnConfig *config.CDNConfig,
	minXMaxY *imagedto.Dimensions,
	img []byte,
	cdn *cdnservice.CdnStruct,
	extension string,
) (downloadedImage []byte, err error) {
	downloadedImage = img
	imagePath := ImageSubPath("", &imageJob.ShopID, imageJob.ProductID, nil, nil, minXMaxY, nil, imageJob.Name)

	if imageJob.ImagesOnCdn != nil {
		if _, ok := (*imageJob.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; ok {
			return downloadedImage, nil
		}
	}

	if len(downloadedImage) == 0 {
		downloadedImage, err = downloadImage(ctx, imageJob.URL)
		if err != nil {
			return nil, fmt.Errorf("could not download image [%v]: %w", imageJob.URL, err)
		}
	}

	output, err := cropImageMinXMaxY(&downloadedImage, minXMaxY, extension)
	if err != nil {
		return downloadedImage, fmt.Errorf("could not minXmaxY crop %v to %vx%v: %w", imagePath, minXMaxY.X, minXMaxY.Y, err)
	}

	contentType := http.DetectContentType(downloadedImage)
	if err = cdn.StoreFile("", path.Join(cdnConfig.ImagesFolder, imagePath), output, contentType); err != nil {
		return downloadedImage, fmt.Errorf("could not store image %v to cdn: %w", imagePath, err)
	}

	return downloadedImage, nil
}

func handleMinYMaxXImage(
	ctx context.Context,
	imageJob *ImageJob,
	cdnConfig *config.CDNConfig,
	minYMaxX *imagedto.Dimensions,
	img []byte,
	cdn *cdnservice.CdnStruct,
	extension string,
) (downloadedImage []byte, err error) {
	downloadedImage = img
	imagePath := ImageSubPath("", &imageJob.ShopID, imageJob.ProductID, nil, nil, nil, minYMaxX, imageJob.Name)

	if imageJob.ImagesOnCdn != nil {
		if _, ok := (*imageJob.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; ok {
			return downloadedImage, nil
		}
	}

	if len(downloadedImage) == 0 {
		downloadedImage, err = downloadImage(ctx, imageJob.URL)
		if err != nil {
			return nil, fmt.Errorf("could not download image [%v]: %w", imageJob.URL, err)
		}
	}

	output, err := cropImageMinYMaxX(&downloadedImage, minYMaxX, extension)
	if err != nil {
		return downloadedImage, fmt.Errorf("could not minXmaxY crop %v to %vx%v: %w", imagePath, minYMaxX.X, minYMaxX.Y, err)
	}

	contentType := http.DetectContentType(downloadedImage)
	if err = cdn.StoreFile("", path.Join(cdnConfig.ImagesFolder, imagePath), output, contentType); err != nil {
		return downloadedImage, fmt.Errorf("could not store image %v to cdn: %w", imagePath, err)
	}

	return downloadedImage, nil
}

func scaleImage(img *[]byte, scaleDimension *int, extension string) (io.ReadSeeker, error) {
	origExtension := strings.TrimPrefix(mimetype.Detect(*img).Extension(), ".")

	if extension == "" {
		extension = origExtension
	}

	src, err := decodeImage(img, origExtension)
	if err != nil {
		return nil, err
	}

	x, y, err := calculateTargetDimensions(scaleDimension, nil, src.Bounds().Max.X, src.Bounds().Max.Y)
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, x, y))

	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	var output bytes.Buffer
	if err := encodeImage(dst, extension, &output); err != nil {
		return nil, err
	}

	return bytes.NewReader(output.Bytes()), nil
}

func cropImage(img *[]byte, cropDimension *imagedto.Dimensions, extension string) (io.ReadSeeker, error) {
	origExtension := strings.TrimPrefix(mimetype.Detect(*img).Extension(), ".")

	if extension == "" {
		extension = origExtension
	}

	src, err := decodeImage(img, origExtension)
	if err != nil {
		return nil, err
	}

	backgroundFillColour := calculateBackgroundColour(src)

	x, y, err := calculateTargetDimensions(nil, cropDimension, src.Bounds().Max.X, src.Bounds().Max.Y)
	if err != nil {
		return nil, err
	}

	shrunkImage := image.NewRGBA(image.Rect(0, 0, x, y))

	draw.NearestNeighbor.Scale(shrunkImage, shrunkImage.Rect, src, src.Bounds(), draw.Over, nil)

	container := image.Rectangle{
		Min: image.Point{X: (cropDimension.X - x) / 2, Y: (cropDimension.Y - y) / 2},
		Max: image.Point{X: cropDimension.X + (cropDimension.X-x)/2, Y: cropDimension.Y + (cropDimension.Y-y)/2},
	}
	result := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: cropDimension.X, Y: cropDimension.Y},
	}
	res := image.NewNRGBA(result)

	if extension != pngExtension {
		for resY := res.Bounds().Min.Y; resY < res.Bounds().Max.Y; resY++ {
			for resX := res.Bounds().Min.X; resX < res.Bounds().Max.X; resX++ {
				res.Set(resX, resY, backgroundFillColour)
			}
		}
	}

	draw.Draw(res, container, shrunkImage, image.Point{}, draw.Over)

	var output bytes.Buffer
	if err := encodeImage(res, extension, &output); err != nil {
		return nil, err
	}

	return bytes.NewReader(output.Bytes()), nil
}

func cropImageMinXMaxY(img *[]byte, minXMaxY *imagedto.Dimensions, extension string) (io.ReadSeeker, error) {
	origExtension := strings.TrimPrefix(mimetype.Detect(*img).Extension(), ".")

	if extension == "" {
		extension = origExtension
	}

	src, err := decodeImage(img, origExtension)
	if err != nil {
		return nil, err
	}

	rgba := calculateBackgroundColour(src)

	x, y, err := calculateMinXMaxYDimensions(minXMaxY, src.Bounds().Max.X, src.Bounds().Max.Y)
	if err != nil {
		return nil, err
	}

	shrunkImage := image.NewRGBA(image.Rect(0, 0, x, y))

	draw.NearestNeighbor.Scale(shrunkImage, shrunkImage.Rect, src, src.Bounds(), draw.Over, nil)

	result := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: x, Y: y},
	}

	if result.Max.X < minXMaxY.X {
		result.Max.X = minXMaxY.X
	}

	container := image.Rectangle{
		Min: image.Point{X: (result.Max.X - x) / 2, Y: 0},
		Max: image.Point{X: result.Max.X + (result.Max.X-x)/2, Y: y},
	}

	res := image.NewNRGBA(result)

	if extension != pngExtension {
		for resY := res.Bounds().Min.Y; resY < res.Bounds().Max.Y; resY++ {
			for resX := res.Bounds().Min.X; resX < res.Bounds().Max.X; resX++ {
				res.Set(resX, resY, rgba)
			}
		}
	}

	draw.Draw(res, container, shrunkImage, image.Point{}, draw.Over)

	var output bytes.Buffer
	if err := encodeImage(res, extension, &output); err != nil {
		return nil, err
	}

	return bytes.NewReader(output.Bytes()), nil
}

func cropImageMinYMaxX(img *[]byte, minYMaxX *imagedto.Dimensions, extension string) (io.ReadSeeker, error) {
	origExtension := strings.TrimPrefix(mimetype.Detect(*img).Extension(), ".")

	if extension == "" {
		extension = origExtension
	}

	src, err := decodeImage(img, origExtension)
	if err != nil {
		return nil, err
	}

	rgba := calculateBackgroundColour(src)

	x, y, err := calculateMinYMaxXDimensions(minYMaxX, src.Bounds().Max.X, src.Bounds().Max.Y)
	if err != nil {
		return nil, err
	}

	shrunkImage := image.NewRGBA(image.Rect(0, 0, x, y))

	draw.NearestNeighbor.Scale(shrunkImage, shrunkImage.Rect, src, src.Bounds(), draw.Over, nil)

	result := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: x, Y: y},
	}

	if result.Max.Y < minYMaxX.Y {
		result.Max.Y = minYMaxX.Y
	}

	container := image.Rectangle{
		Min: image.Point{X: 0, Y: (result.Max.Y - y) / 2},
		Max: image.Point{X: x, Y: result.Max.Y + (result.Max.Y-y)/2},
	}

	res := image.NewNRGBA(result)

	if extension != pngExtension {
		for resY := res.Bounds().Min.Y; resY < res.Bounds().Max.Y; resY++ {
			for resX := res.Bounds().Min.X; resX < res.Bounds().Max.X; resX++ {
				res.Set(resX, resY, rgba)
			}
		}
	}

	draw.Draw(res, container, shrunkImage, image.Point{}, draw.Over)

	var output bytes.Buffer
	if err := encodeImage(res, extension, &output); err != nil {
		return nil, err
	}

	return bytes.NewReader(output.Bytes()), nil
}

func decodeImage(img *[]byte, extension string) (image.Image, error) {
	switch extension {
	case jpgExtension, jpegExtension:
		return jpeg.Decode(bytes.NewReader(*img))
	case pngExtension:
		return png.Decode(bytes.NewReader(*img))
	case webpExtension:
		return webp.Decode(bytes.NewReader(*img))
	default:
		return nil, fmt.Errorf("%w : %v", errUnsupportedFileType, extension)
	}
}

func encodeImage(img image.Image, extension string, output *bytes.Buffer) error {
	switch extension {
	case jpgExtension, jpegExtension:
		return jpeg.Encode(output, img, nil)
	case pngExtension:
		return png.Encode(output, img)
	default:
		return fmt.Errorf("%w : %v", errUnsupportedFileType, extension)
	}
}

// calculateBackgroundColour returns the colour that the image has only if 3 corners have the same one.
// Otherwise, it returns white since we are not sure whether it's a background colour or the photo takes up 2 corners.
func calculateBackgroundColour(img image.Image) color.RGBA {
	type rgbaStruct struct{ r, g, b, a uint32 }

	points := [][]int{
		{0, 0},
		{img.Bounds().Max.X - 1, 0},
		{0, img.Bounds().Max.Y - 1},
		{img.Bounds().Max.X - 1, img.Bounds().Max.Y - 1},
	}
	rgbaMap := make(map[rgbaStruct]int)

	for _, p := range points {
		rgba := rgbaStruct{}
		rgba.r, rgba.g, rgba.b, rgba.a = img.At(p[0], p[1]).RGBA()
		rgbaMap[rgba] += 1
	}

	for rgba, number := range rgbaMap {
		if number > 2 {
			return color.RGBA{R: uint8(rgba.r >> 8), G: uint8(rgba.g >> 8), B: uint8(rgba.b >> 8), A: uint8(rgba.a >> 8)}
		}
	}

	return color.RGBA{R: uint8(255), G: uint8(255), B: uint8(255), A: uint8(255)}
}

func calculateMinXMaxYDimensions(minXMaxY *imagedto.Dimensions, srcX, srcY int) (x, y int, err error) {
	if minXMaxY == nil || (minXMaxY.X == 0 && minXMaxY.Y == 0) {
		return x, y, errNoDimensionsDefined
	}

	y = minXMaxY.Y
	div := float64(srcY) / float64(y)
	x = int(float64(srcX) / div)

	return x, y, nil
}

func calculateMinYMaxXDimensions(minYMaxX *imagedto.Dimensions, srcX, srcY int) (x, y int, err error) {
	if minYMaxX == nil || (minYMaxX.X == 0 && minYMaxX.Y == 0) {
		return x, y, errNoDimensionsDefined
	}

	x = minYMaxX.X
	div := float64(srcX) / float64(x)
	y = int(float64(srcY) / div)

	return x, y, nil
}

func calculateTargetDimensions(
	scaleDimension *int,
	cropDimensions *imagedto.Dimensions,
	srcX,
	srcY int,
) (x, y int, err error) {
	if cropDimensions != nil {
		scaleDimension = &cropDimensions.X
		if cropDimensions.Y > cropDimensions.X {
			scaleDimension = &cropDimensions.Y
		}
	}

	if scaleDimension != nil {
		var div float64

		if srcX > srcY {
			x = *scaleDimension
			div = float64(srcX) / float64(*scaleDimension)
			y = int(float64(srcY) / div)
		} else {
			y = *scaleDimension
			div = float64(srcY) / float64(*scaleDimension)
			x = int(float64(srcX) / div)
		}

		return x, y, nil
	}

	return x, y, errNoDimensionsDefined
}

func downloadImage(parentContext context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parentContext, 20*time.Second)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
