# spaceapi

## Usage

### Sensors
    https://<server>/sensors/<sensortype>/<location>/<value>[/<unit>[/<description>]]

    curl -XPUT -d '[                        =99% v52%
        {
            "SensorType": "temperature",
            "Location": "lounge",
            "Value": 10.0,
            "Unit": "C",
            "Description": "test"
        }
    ]' https://<server>/sensors/


### Door
    https://<server>/door/<state>/<users>/<usernames>/
