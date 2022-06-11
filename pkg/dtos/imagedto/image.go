package imagedto

type UploadLogoResp struct {
	URL string `json:"url"`
}

type UpdateImagesReq struct {
	ID       int   `json:"id"`
	Carousel *bool `json:"carousel"`
	Gallery  *bool `json:"gallery"`
}
