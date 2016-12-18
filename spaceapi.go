package main

import (
    "fmt"
    "flag"
    "net/http"
    "net/http/fcgi"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/sqlite"
    "log"
    "strings"
    "strconv"
    "encoding/json"
    "io/ioutil"
    "database/sql"
    "os"
)

type SensorTypes struct{
    gorm.Model
    Name string     `sql:"size:255;unique;index"`
}

type Sensor struct{
    gorm.Model
    Type SensorTypes `gorm:"ForeignKey:TypeRefer"`
    TypeRefer int
    Location string
    Value float32
    Description sql.NullString
    Unit sql.NullString
}

var err error
var db *gorm.DB
var local = flag.String("local", "", "serve as webserver, example: 0.0.0.0:8000")

func main() {
    f, err := os.OpenFile("spaceapi.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    log.SetOutput(f)
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    initDatabase()
    flag.Parse()
    h := http.NewServeMux()
    h.HandleFunc("/", handler)
    if *local != "" {
        http.ListenAndServe(":8080", h)
    } else {
        err = fcgi.Serve(nil, h)
        if err != nil {
            log.Fatal(err)
        }
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    log.Printf(r.URL.Path)
    arr := strings.Split(r.URL.Path[1:],"/")
    if len(arr) >= 3 {
        f, err := strconv.ParseFloat(arr[2], 32)
        if err == nil {
            switch len(arr) {
                case 3:
                    updateOrInsertValue(arr[0], arr[1], float32(f), "", "")
                case 4:
                    updateOrInsertValue(arr[0], arr[1], float32(f), arr[3], "")
                case 5:
                    updateOrInsertValue(arr[0], arr[1], float32(f), arr[3], arr[4])
                default:
                    fmt.Fprintf(w, "NOK")
                    return
            }
            fmt.Fprintf(w, "Ok")
            return
        }
        fmt.Fprintf(w, "NOK")
        return
    }
    headers := w.Header()
    headers.Add("Content-Type", "application/json; charset=utf-8")
    file, e := ioutil.ReadFile("./spaceapi.json")
    if e != nil {
        log.Fatal(e)
    }

    var f interface{}
    err := json.Unmarshal(file, &f)
    if err != nil {
        log.Fatal(err)
    }
    m := f.(map[string]interface{})
    state := m["state"].(map[string]interface{})

    state["lastchange"] = GetLastUpdate()
    sensor := make(map[string]interface{})

    types := getSensorTypes()
    fmt.Println(len(types))
    for i := range types{
        s := getSensorsByType(types[i].Model.ID)
        sensorType := make([]map[string]interface{},len(s))
        //sensorType := make([]Sensor,len(s))
        for j := range s{
            curSensor := make(map[string]interface{})
            curSensor["location"] = s[j].Location
            curSensor["value"] = s[j].Value
            if s[j].Description.Valid {
                curSensor["description"] = s[j].Description.String
            }
            if s[j].Unit.Valid {
                curSensor["unit"] = s[j].Unit.String
            }
            sensorType[j] = curSensor
            //sensorType[j] = s[j]
        }
        sensor[types[i].Name] = sensorType
    }

    state["sensors"] = sensor
    bytes, err := json.MarshalIndent(f, "", "\t")
    if err != nil {
        log.Fatal(err)
    }
    _, err = w.Write(bytes)
    if err != nil {
        log.Fatal(err)
    }
}


func getSensorTypes() []SensorTypes{
    var arr []SensorTypes
    db.Find(&arr)
    if db.Error != nil {
        log.Fatal("bla err")
    }
    return arr
}

func getSensorsByType(id uint) []Sensor{
    var arr []Sensor
    db.Find(&arr, Sensor{TypeRefer: int(id)})
    return arr
}


func initDatabase(){

    db, err = gorm.Open("sqlite3", "./foo.db")
    if err != nil {
        log.Fatal(err)
    }
    db.AutoMigrate(&SensorTypes{})
    db.AutoMigrate(&Sensor{})

}

func GetLastUpdate() int64{
    var s Sensor
    db.Order("UpdatedAt").First(&s)
    if db.Error != nil {
        return 0
    }
    return s.UpdatedAt.Unix()
}


func updateOrInsertValue(sensortype string, location string, value float32, unit string, desc string){
    var sType SensorTypes

    fmt.Println("update")
    db.FirstOrCreate(&sType, SensorTypes{Name: sensortype})

    fmt.Println("update %s", sType.Name)

    var sensor Sensor

    db.FirstOrCreate(&sensor, Sensor{TypeRefer: int(sType.ID), Location: location})

    if len(unit) > 0 {
        sensor.Unit = sql.NullString{unit, true}
    }
    if len(desc) > 0 {
        sensor.Description = sql.NullString{desc, true}
    }
    sensor.Value = value
    db.Save(&sensor)
}

