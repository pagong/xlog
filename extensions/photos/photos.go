package photos

import (
	"embed"
	"fmt"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/emad-elsaid/types"
	"github.com/emad-elsaid/xlog"
	"github.com/emad-elsaid/xlog/extensions/shortcode"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

//go:embed templates
var templates embed.FS

var supportedExt = types.Slice[string]{".jpg", ".jpeg", ".gif", ".png"}

func init() {
	shortcode.ShortCode("photos", photosShortcode)
	xlog.RegisterTemplate(templates, "templates")
	xlog.Get(`/\+/photos/thumbnail/{path:.*}`, resizeHandler)
	xlog.Get(`/\+/photos/photo/{path:.*}`, photoHandler)
}

type Photo struct {
	Thumbnail string
	Page      string
	Original  string
	Width     int
	Height    int
	Exif      *exif.Exif
	Time      time.Time
}

func (p *Photo) Name() string {
	base := path.Base(p.Thumbnail)
	ext := path.Ext(base)
	return base[:len(base)-len(ext)]
}

func (p *Photo) Camera() string {
	out := ""

	make, err := p.Exif.Get(exif.Make)
	if err == nil {
		out, _ = make.StringVal()
	}

	camModel, err := p.Exif.Get(exif.Model)
	if err == nil {
		str, _ := camModel.StringVal()
		out += " " + str
	}

	return out
}

func (p *Photo) FocalLength() string {
	focal, err := p.Exif.Get(exif.FocalLength)
	if err != nil {
		return ""
	}

	nom, denom, err := focal.Rat2(0)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%dmm", nom/denom)
}

func (p *Photo) Aperture() string {
	aperture, err := p.Exif.Get(exif.ApertureValue)
	if err != nil {
		return ""
	}

	anom, adenom, err := aperture.Rat2(0)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("f/%.1f", float32(anom)/float32(adenom))
}

func (p *Photo) ShutterSpeed() string {
	shutter, err := p.Exif.Get(exif.ShutterSpeedValue)
	if err != nil {
		return ""
	}

	snom, sdenom, err := shutter.Rat2(0)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("1/%.0fs", math.Pow(2, float64(snom)/float64(sdenom)))
}

func (p *Photo) ISO() string {
	iso, err := p.Exif.Get(exif.ISOSpeedRatings)
	if err != nil {
		return ""
	}

	return iso.String()
}

func (p *Photo) Lens() string {
	output := ""
	make, err := p.Exif.Get(exif.LensMake)
	if err == nil {
		output, _ = make.StringVal()
	}

	model, err := p.Exif.Get(exif.LensModel)
	if err == nil {
		val, err := model.StringVal()
		if err == nil {
			output += val
		}
	}

	return output
}

func (p *Photo) AspectRatio() float32 {
	return float32(p.Width) / float32(p.Height)
}

func NewPhoto(path string) (*Photo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	exifData, _ := exif.Decode(f)
	t := stat.ModTime()

	if exifData != nil {
		shootingTime, err := exifData.DateTime()
		if err == nil {
			t = shootingTime
		}
	}

	f.Seek(0, os.SEEK_SET)
	img, _, err := image.Decode(f)
	width, height := 0, 0
	if err == nil {
		max := img.Bounds().Max
		width = max.X
		height = max.Y
	}

	return &Photo{
		Thumbnail: "/+/photos/thumbnail/" + path,
		Page:      "/+/photos/photo/" + path,
		Original:  path,
		Width:     width,
		Height:    height,
		Exif:      exifData,
		Time:      t,
	}, nil
}

func photosShortcode(input xlog.Markdown) template.HTML {
	p := strings.TrimSpace(string(input))

	photos := []*Photo{}

	err := filepath.WalkDir(p, func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type().IsRegular() && supportedExt.Include(strings.ToLower(path.Ext(file))) {
			photo, err := NewPhoto(file)
			if err != nil {
				return err
			}

			xlog.RegisterBuildPage(photo.Thumbnail, false)
			xlog.RegisterBuildPage(photo.Page, true)
			photos = append(photos, photo)
		}

		return nil
	})

	if err != nil {
		return template.HTML(err.Error())
	}

	sort.Slice(photos, func(i, j int) bool {
		return photos[i].Time.After(photos[j].Time)
	})

	return xlog.Partial("photos", xlog.Locals{
		"photos": photos,
	})
}

func resizeHandler(w xlog.Response, r xlog.Request) xlog.Output {
	vars := xlog.Vars(r)
	path := vars["path"]

	return func(w xlog.Response, r xlog.Request) {
		inputImage, err := os.Open(path)
		if err != nil {
			fmt.Fprint(w, err.Error())
			return
		}
		defer inputImage.Close()

		src, _, _ := image.Decode(inputImage)
		dim := src.Bounds().Max

		width := 700
		height := int(float32(width) / float32(dim.X) * float32(dim.Y))

		dst := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

		png.Encode(w, dst)
	}
}

func photoHandler(w xlog.Response, r xlog.Request) xlog.Output {
	vars := xlog.Vars(r)
	path := vars["path"]
	photo, err := NewPhoto(path)
	if err != nil {
		return xlog.InternalServerError(err)
	}

	return xlog.Render("photo", xlog.Locals{
		"photo": photo,
		"title": photo.Name(),
	})
}
