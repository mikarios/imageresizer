package imageservice_test

import (
	"testing"

	"github.com/mikarios/golib/pointers"

	"github.com/mikarios/imageresizer/internal/imagehelper"
	"github.com/mikarios/imageresizer/pkg/dtos/imagedto"
)

// nolint:funlen // yes, it's big...
func Test_createImageSubPath(t *testing.T) {
	t.Parallel()

	type args struct {
		prefix         string
		shopID         *int
		scaleDimension *int
		cropDimensions *imagedto.Dimensions
		minXMaxY       *imagedto.Dimensions
		minYMaxX       *imagedto.Dimensions
		fileName       string
		productID      string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "nothing",
			args: args{},
			want: "",
		},
		{
			name: "only prefix",
			args: args{
				prefix: "myprefix",
			},
			want: "myprefix",
		},
		{
			name: "only shopID",
			args: args{
				shopID: pointers.Ptr(0),
			},
			want: "0",
		},
		{
			name: "only filename",
			args: args{
				fileName: "file.name",
			},
			want: "file.name",
		},
		{
			name: "only scaleDimension",
			args: args{
				scaleDimension: pointers.Ptr(1000),
			},
			want: "1000",
		},
		{
			name: "only crop dimensions",
			args: args{
				cropDimensions: &imagedto.Dimensions{
					X: 100,
					Y: 100,
				},
			},
			want: "100x100",
		},
		{
			name: "all with scaleDimension",
			args: args{
				prefix:         "prefix",
				productID:      "asd",
				shopID:         pointers.Ptr(1),
				scaleDimension: pointers.Ptr(1000),
				fileName:       "file.name",
			},
			want: "prefix/1/asd/1000/file.name",
		},
		{
			name: "all with cropDimensions",
			args: args{
				prefix:    "prefix",
				shopID:    pointers.Ptr(1),
				productID: "asd",
				cropDimensions: &imagedto.Dimensions{
					X: 100,
					Y: 100,
				},
				fileName: "file.name",
			},
			want: "prefix/1/asd/100x100/file.name",
		},
		{
			name: "minXMaxY",
			args: args{
				prefix:    "prefix",
				shopID:    pointers.Ptr(1),
				productID: "asd",
				minXMaxY: &imagedto.Dimensions{
					X: 100,
					Y: 100,
				},
				fileName: "file.name",
			},
			want: "prefix/1/asd/minxmaxy/100x100/file.name",
		},
		{
			name: "minYMaxX",
			args: args{
				prefix:    "prefix",
				shopID:    pointers.Ptr(1),
				productID: "asd",
				minYMaxX: &imagedto.Dimensions{
					X: 100,
					Y: 100,
				},
				fileName: "file.name",
			},
			want: "prefix/1/asd/minymaxx/100x100/file.name",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := imagehelper.ImageSubPath(
				tt.args.prefix,
				tt.args.shopID,
				tt.args.productID,
				tt.args.scaleDimension,
				tt.args.cropDimensions,
				tt.args.minXMaxY,
				tt.args.minYMaxX,
				tt.args.fileName,
			); got != tt.want {
				t.Errorf("imageSubPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func Test_processJobImage(t *testing.T) {
//	cdn := cdnservice.Init()
//	cfg := config.Init("")
//	job := imageJob{
//		ImageStruct: &imagedto.ImageStruct{
//			URL: "https://manos.fra1.digitaloceanspaces.com/IMG_2696.jpg",
//			// ScaleDimensionMax: []*int{tools.Int(100), tools.Int(600)},
//			CropDimensions: []*imagedto.Dimensions{{1000, 1000}, {1500, 1500}},
//			MinXMaxY:       []*imagedto.Dimensions{{1000, 1000}, {1500, 1500}},
//			MinYMaxX:       []*imagedto.Dimensions{{1000, 1000}, {1500, 1500}},
//			Name:           "IMG_2696.jpg",
//			ProductID:      "product2",
//		},
//		ShopID:    100,
//		errorChan: nil,
//	}
//
//	start := time.Now()
//	baseImagePath := ImageSubPath("", &job.ShopID, "", nil, nil, nil, nil, "")
//	listOfFiles, err := cdn.ListFilesToMap("", path.Join(cfg.CDN.ImagesFolder, baseImagePath))
//	if err != nil {
//		listOfFiles = nil
//	}
//	logger.Debug(
//		context.Background(),
//		fmt.Sprintf("LIST: %v finished. Took: %v", baseImagePath, time.Since(start)),
//	)
//
//	job.ImagesOnCdn = listOfFiles
//
//	err = processJobImage(context.Background(), &job)
//	if err != nil {
//		t.Errorf("failed %v", err)
//	}
// }
