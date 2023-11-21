package filtered_camera

import (
	"context"
	"fmt"
	"image"
	"sort"
	"strings"
	"regexp"
	"sync"
	"time"

	"github.com/edaniels/golog"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/data"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
	"go.viam.com/rdk/services/vision"
	"go.viam.com/rdk/vision/classification"
	"go.viam.com/rdk/vision/objectdetection"

	"github.com/viamrobotics/gostream"
	"go.viam.com/utils"
)

var Model = resource.ModelNamespace("erh").WithFamily("camera").WithModel("filtered-camera")

type Config struct {
	Camera        string
	Vision        string
	WindowSeconds int `json:"window_seconds"`

	Classifications map[string]float64
	Objects         map[string]float64
}

func (cfg *Config) keepClassifications(cs []classification.Classification) bool {
	for _, c := range cs {
		if cfg.keepClassification(c) {
			return true
		}
	}
	return false
}

func (cfg *Config) keepClassification(c classification.Classification) bool {
	min, has := cfg.Classifications[c.Label()]
	if has && c.Score() > min {
		return true
	}

	min, has = cfg.Classifications["*"]
	if has && c.Score() > min {
		return true
	}
	// check for substring match
	for key, _ := range cfg.Classifications {
		if strings.Contains(key, "*") {
			if match, err := regexp.MatchString(key, c.Label()); match {
				if err != nil {
					continue
				}
				return true
			}
		}
	}

	return false
}

func (cfg *Config) keepObjects(ds []objectdetection.Detection) bool {
	for _, d := range ds {
		if cfg.keepObject(d) {
			return true
		}
	}

	return false
}

func (cfg *Config) keepObject(d objectdetection.Detection) bool {
	min, has := cfg.Objects[d.Label()]
	if has && d.Score() > min {
		return true
	}

	min, has = cfg.Objects["*"]
	if has && d.Score() > min {
		return true
	}
	// check for substring match
	for key, _ := range cfg.Objects {
		if strings.Contains(key, "*") {
			if match, err := regexp.MatchString(key, d.Label()); match {
				if err != nil {
					continue
				}
				return true
			}
		}
	}
	return false
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

			fc := &filteredCamera{name: conf.ResourceName(), conf: newConf, logger: logger}

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

type cachedData struct {
	imgs []camera.NamedImage
	meta resource.ResponseMetadata
}

type filteredCamera struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	conf   *Config
	logger golog.Logger

	cam camera.Camera
	vis vision.Service

	mu          sync.Mutex
	buffer      []cachedData
	toSend      []cachedData
	captureTill time.Time
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

	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.addToBuffer_inlock(images, meta)

	if len(fc.toSend) > 0 {
		x := fc.toSend[0]
		fc.toSend = fc.toSend[1:]
		return x.imgs, x.meta, nil
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

	fs.fc.mu.Lock()
	defer fs.fc.mu.Unlock()

	fs.fc.addToBuffer_inlock([]camera.NamedImage{{img, ""}}, resource.ResponseMetadata{CapturedAt: time.Now()})

	return nil, nil, data.ErrNoCaptureToStore
}

func (fc *filteredCamera) addToBuffer_inlock(imgs []camera.NamedImage, meta resource.ResponseMetadata) {
	if fc.conf.WindowSeconds == 0 {
		return
	}

	fc.cleanBuffer_inlock()
	fc.buffer = append(fc.buffer, cachedData{imgs, meta})
}

func (fs filterStream) Close(ctx context.Context) error {
	return fs.cameraStream.Close(ctx)
}

func (fc filteredCamera) windowDuration() time.Duration {
	return time.Second * time.Duration(fc.conf.WindowSeconds)
}

func (fc *filteredCamera) cleanBuffer_inlock() {
	sort.Slice(fc.buffer, func(i, j int) bool {
		a := fc.buffer[i]
		b := fc.buffer[j]
		return a.meta.CapturedAt.Before(b.meta.CapturedAt)
	})

	early := time.Now().Add(-1 * fc.windowDuration())
	for len(fc.buffer) > 0 {
		if fc.buffer[0].meta.CapturedAt.After(early) {
			return
		}
		fc.buffer = fc.buffer[1:]
	}
}

func (fc *filteredCamera) markShouldSend() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.captureTill = time.Now().Add(fc.windowDuration())
	fc.cleanBuffer_inlock()

	for _, x := range fc.buffer {
		fc.toSend = append(fc.toSend, x)
	}

	fc.buffer = []cachedData{}
}

func (fc *filteredCamera) shouldSend(ctx context.Context, img image.Image) (bool, error) {

	if len(fc.conf.Classifications) > 0 {
		res, err := fc.vis.Classifications(ctx, img, 100, nil)
		if err != nil {
			return false, err
		}

		if fc.conf.keepClassifications(res) {
			fc.logger.Infof("keeping image with classifications %v", res)
			fc.markShouldSend()
			return true, nil
		}
	}

	if len(fc.conf.Objects) > 0 {
		res, err := fc.vis.Detections(ctx, img, nil)
		if err != nil {
			return false, err
		}

		if fc.conf.keepObjects(res) {
			fc.logger.Infof("keeping image with objects %v", res)
			fc.markShouldSend()
			return true, nil
		}
	}

	if time.Now().Before(fc.captureTill) {
		// send, but don't update captureTill
		return true, nil
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
