Viam filtered camera module
======

This module let's you only sync data to the viam cloud that you want.
You can say either classify an image and only sync some, or look for object in an image and sync ones that have a certain object.

https://app.viam.com/module/erh/filtered-camera

### Example Config

```
"name": "filter",
"model": "erh:camera:filtered-camera",
"type": "camera",
"namespace": "rdk",
"attributes" : {
  "camera": "my-cam",
  "vision": "my-vision-service",
  "classifications": {
    "COUNTDOWN*": 0.6,
    "ALARM": 0.5
  },
  "detections": {
   "*": 0.85
  }
}
```
For example, this config would save all images with a classification label that exactly matched "ALARM" with greater than 0.5 confidence, or partially matched "COUNTDOWN", e.g. "COUNTDOWN: 10 s remain!" with greater than 0.6 confidence. It would also save all images that had any detection with a confidence above 0.85.  


