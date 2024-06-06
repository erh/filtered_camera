# Viam filtered camera module

If your smart machine captures a lot of data, you can filter captured data to selectively store only the data that meets certain criteria.
This module allows you to filter image data by:

- Specified classification labels and required confidence scores
- Detected objects and their associated required confidence scores

This allows you to:
- Classify images and only sync images that have the required label
- Look for objects in an image and sync the images that have a certain object

This module also allows you to specifcy a time window for syncing the data captured in the N seconds before the capture criteria were met.

## Configure your filtered camera

> [!NOTE]  
> Before configuring your camera, you must [create a robot](https://docs.viam.com/manage/fleet/robots/#add-a-new-robot).

Navigate to the **Config** tab of your robot’s page in [the Viam app](https://app.viam.com/). Click on the **Components** subtab and click **Create component**. Select the `camera` type, then select the [`filtered-camera` model](https://app.viam.com/module/erh/filtered-camera) and click **Add module**. Enter a name for your camera and click **Create**.

On the new component panel, copy and paste the following attribute template into your base’s **Attributes** box. 

```json
{
    "camera": "<your_camera_name>",
    "vision": "<your_vision_service_name>",
    "window_seconds": <time_window_for_capture>,
    "classifications": {
        "<label>": <confidence_score>,
        "<label>": <confidence_score>
    },
    "objects": {
        "<label>": <confidence_score>,
        "<label>": <confidence_score>
    }
}
```

> [!NOTE]  
> For more information, see [Configure a Robot](https://docs.viam.com/manage/configuration/).

### Attributes

The following attributes are available for `erh:camera:filtered-camera`:

| Name | Type | Inclusion | Description |
| ---- | ------ | ------------ | ----------- |
| `camera` | string | **Required** | The name of the camera to filter images for. |
| `vision` | string | **Required** | The vision service used for image classifications or detections. |
| `window_seconds` | float64 | Optional | The size of the time window during which images are buffered. When a condition is met, a confidence score for a detection/classification exceeds the required confidence score, the buffered images are stored, allowing us to see the photos taken in the N number of seconds preceding the condition being met.
in seconds for capturing and sending images. |
| `classifications` | float64 | Optional | A map of
classification labels and the confidence scores required for filtering. |
| `objects` | float64 | Optional | A map of object detection
labels and the confidence scores required for filtering. |

### Example configurations:

```json
{
    "camera": "my_camera",
    "vision": "my_red_car_detector",
    "windowSeconds": 5,
    "objects": {
        "red_car": 0.6
    }
}
```

## Local Development

To use the `filtered_camera` module, clone this repository to your
machine’s computer, navigate to the `module` directory, and run:

```go
go build
```

On your robot’s page in the [Viam app](https://app.viam.com/), enter
the [module’s executable
path](/registry/create/#prepare-the-module-for-execution, then click
**Add module**.
The name must use only lowercase characters.
Then, click **Save config**.

## Next Steps

- To test your camera, go to the [**Control** tab](https://docs.viam.com/manage/fleet/robots/#control).
- To test that your camera's images are filtering correctly, [go to the **Data** tab](https://docs.viam.com/data/view/).

## License
Copyright 2021-2023 Viam Inc. <br>
Apache 2.0
