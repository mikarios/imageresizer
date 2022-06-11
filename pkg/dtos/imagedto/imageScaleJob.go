package imagedto

import (
	"github.com/streadway/amqp"
)

const (
	PriorityUrgent priorityType = "urgent"
	PriorityNormal priorityType = "normal"
)

type priorityType string

type ImageScaleJobReq struct {
	Job      *ImageProcessJobData `json:"job"`
	Priority priorityType         `json:"priority"`
}

type ImageProcessJob struct {
	Data     *ImageProcessJobData
	QueueJob amqp.Delivery
}

type ImageProcessJobData struct {
	ShopID         int            `json:"shopID"`
	ImageExtension string         `json:"imageExtension"`
	Images         []*ImageStruct `json:"images"`
	DeleteImages   []string       `json:"deleteImages"`
}

// ImageStruct holds the information on how the image will be scaled and where it will be stored.
// Only one of ScaleDimensionMax, CropDimensions should have value.
// The image will be stored in the following folder structure:
// If ScaleDimensionMax is set: /<shopID>/<ScaleDimensionMax>/<Name>
// If CropDimensions is set: /<shopID>/<CropDimensions.X>x<CropDimensions.Y>/<Name>
// If MinXMaxY is set: /<shopID>/minxmaxy/<MinXMaxY.X>x<MinXMaxY.Y>/<Name>
// If MinYMaxX is set: /<shopID>/miny<MinYMaxX.X>maxx<MinYMaxX.Y>/<Name>
// URL is the url from which the image will be downloaded
// Name is the filename.
type ImageStruct struct {
	URL               string        `json:"url"`
	ScaleDimensionMax []*int        `json:"scaleDimensionMax,omitempty"`
	CropDimensions    []*Dimensions `json:"cropDimensions,omitempty"`
	MinXMaxY          []*Dimensions `json:"minXMaxY"`
	MinYMaxX          []*Dimensions `json:"minYMaxX"`
	Name              string        `json:"name,omitempty"`
	ProductID         string        `json:"productID,omitempty"`
}

type Dimensions struct {
	X int `json:"x"`
	Y int `json:"y"`
}
