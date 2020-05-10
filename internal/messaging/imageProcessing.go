package messaging

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	core "github.com/miphilipp/devchat-server/internal"
	"github.com/nfnt/resize"
)

func resizeImage(reader io.Reader, writer io.Writer, maxWidth, maxHeigth uint) (error, core.Size) {
	img, imageType, err := image.Decode(reader)
	if err != nil {
		return err, core.Size{}
	}

	thumbnail := resize.Thumbnail(maxWidth, maxHeigth, img, resize.Bicubic)
	switch imageType {
	case "png":
		err = png.Encode(writer, thumbnail)
	case "jpeg":
		err = jpeg.Encode(writer, thumbnail, nil)
	case "gif":
		err = gif.Encode(writer, thumbnail, nil)
	default:
		return core.ErrInvalidFileType, core.Size{}
	}

	if err != nil {
		return err, core.Size{}
	}

	return nil, core.Size{
		Width:  thumbnail.Bounds().Dx(),
		Height: thumbnail.Bounds().Dy(),
	}
}
