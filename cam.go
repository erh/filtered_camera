package filtered_camera

import (
	"context"

	"github.com/edaniels/golog"
	//"github.com/edaniels/gostream"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/resource"
)

var Model = resource.ModelNamespace("erh").WithFamily("camera").WithModel("filtered_camera")

type Config struct {
	Camera string
	Vision string
}

func init() {
	resource.RegisterComponent(camera.API, Model, resource.Registration[camera.Camera, *Config]{
		Constructor: func(ctx context.Context, _ resource.Dependencies, conf resource.Config, logger golog.Logger) (camera.Camera, error) {
			newConf, err := resource.NativeConfig[*Config](conf)
			if err != nil {
				return nil, err
			}
			return NewFilteredCamera(ctx, conf.ResourceName(), newConf, logger)
		},
	})
}

func NewFilteredCamera(ctx context.Context, name resource.Name, conf *Config, logger golog.Logger) (camera.Camera, error) {
	panic(1)
}

