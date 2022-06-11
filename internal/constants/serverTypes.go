package constants

type ServerType string

var (
	ServerTypes = struct {
		None         ServerType
		ImageResizer ServerType
	}{
		None:         "none",
		ImageResizer: "imageresizer",
	}

	ServerTypeList = []ServerType{
		ServerTypes.ImageResizer,
	}
)
