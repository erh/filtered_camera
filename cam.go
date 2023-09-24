package filtered_camera

import (
	"context"
	"fmt"
	"image"

	"github.com/edaniels/golog"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/data"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
	"go.viam.com/rdk/services/vision"

	"github.com/viamrobotics/gostream"
	"go.viam.com/utils"
)

var Model = resource.ModelNamespace("erh").WithFamily("camera").WithModel("filtered-camera")

type Config struct {
	Camera string
	Vision string

	Classifications map[string]float64
	Objects         map[string]float64
}

func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Camera == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "camera")
	}

	if cfg.Vision == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "vision")
	}

	return []string{cfg.Camera, cfg.Vision}, nil
}

func init() {
	resource.RegisterComponent(camera.API, Model, resource.Registration[camera.Camera, *Config]{
		Constructor: func(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger golog.Logger) (camera.Camera, error) {
			newConf, err := resource.NativeConfig[*Config](conf)
			if err != nil {
				return nil, err
			}

			fc := &filteredCamera{name: conf.ResourceName(), conf: newConf}

			fc.cam, err = camera.FromDependencies(deps, newConf.Camera)
			if err != nil {
				return nil, err
			}

			fc.vis, err = vision.FromDependencies(deps, newConf.Vision)
			if err != nil {
				return nil, err
			}

			return fc, nil
		},
	})
}

type filteredCamera struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name resource.Name
	conf *Config

	cam camera.Camera
	vis vision.Service
}

func (fc *filteredCamera) Name() resource.Name {
	return fc.name
}

func (fc *filteredCamera) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, resource.ErrDoUnimplemented
}

func (fc *filteredCamera) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	images, meta, err := fc.cam.Images(ctx)
	if err != nil {
		return images, meta, err
	}

	if ctx.Value(data.FromDMContextKey{}) != true {
		return images, meta, nil
	}

	for _, img := range images {
		shouldSend, err := fc.shouldSend(ctx, img.Image)
		if err != nil {
			return nil, meta, err
		}

		if shouldSend {
			return images, meta, nil
		}
	}

	return nil, meta, data.ErrNoCaptureToStore
}

func (fc *filteredCamera) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	camStream, err := fc.cam.Stream(ctx, errHandlers...)
	if err != nil {
		return nil, err
	}

	return filterStream{camStream, fc}, nil
}

type filterStream struct {
	cameraStream gostream.VideoStream
	fc           *filteredCamera
}

func (fs filterStream) Next(ctx context.Context) (image.Image, func(), error) {
	if ctx.Value(data.FromDMContextKey{}) != true {
		// If not data management collector, return underlying stream contents without filtering.
		return fs.cameraStream.Next(ctx)
	}

	img, release, err := fs.cameraStream.Next(ctx)
	if err != nil {
		return nil, nil, err
	}

	should, err := fs.fc.shouldSend(ctx, img)
	if err != nil {
		return nil, nil, err
	}

	if should {
		return img, release, nil
	}

	return nil, nil, data.ErrNoCaptureToStore
}

func (fs filterStream) Close(ctx context.Context) error {
	return fs.cameraStream.Close(ctx)
}

func (fc *filteredCamera) shouldSend(ctx context.Context, img image.Image) (bool, error) {

	if len(fc.conf.Classifications) > 0 {
		res, err := fc.vis.Classifications(ctx, img, 100, nil)
		if err != nil {
			return false, err
		}

		for _, c := range res {
			min, has := fc.conf.Classifications[c.Label()]
			if has && c.Score() > min {
				return true, nil
			}
		}
	}

	if len(fc.conf.Objects) > 0 {
		res, err := fc.vis.Detections(ctx, img, nil)
		if err != nil {
			return false, err
		}

		for _, d := range res {
			min, has := fc.conf.Objects[d.Label()]
			if has && d.Score() > min {
				return true, nil
			}
		}
	}

	return false, nil
}

func (fc *filteredCamera) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	return nil, fmt.Errorf("filteredCamera doesn't support pointclouds yes")
}

func (fc *filteredCamera) Properties(ctx context.Context) (camera.Properties, error) {
	p, err := fc.cam.Properties(ctx)
	if err == nil {
		p.SupportsPCD = false
	}
	return p, err
}

func (fc *filteredCamera) Projector(ctx context.Context) (transform.Projector, error) {
	return fc.cam.Projector(ctx)
}
