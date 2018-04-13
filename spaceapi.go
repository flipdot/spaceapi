package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/fcgi"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type NoIdModel struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type Sensor struct {
	NoIdModel
	Type        string `gorm:"primary_key"`
	Location    string `gorm:"primary_key"`
	Value       float32
	Description sql.NullString
	Unit        sql.NullString
	Name        string `gorm:"primary_key"`
	ApiKey      sql.NullString
}

type Door struct {
	gorm.Model
	Open           bool
	UserNames      sql.NullString `gorm:"type:varchar(255)"`
	UserCount      int
	LastDoorChange time.Time
}

type UpdateParam struct {
	SensorType  string
	Location    string
	Value       float32
	Unit        string
	Description string
	Name        string
}

var err error
var db *gorm.DB
var local = flag.String("local", "", "serve as webserver, example: 0.0.0.0:8000")

func main() {
	f, err := os.OpenFile("spaceapi.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitDatabase(f)
	flag.Parse()
	h := http.NewServeMux()
	h.HandleFunc("/", SpaceapiHandler)
	h.HandleFunc("/sensors/", SensorHandler)
	h.HandleFunc("/door/", DoorHandler)
	if *local != "" {
		log.Println("Listening at", *local)
		log.Fatalln(http.ListenAndServe(*local, h))
	} else {
		err = fcgi.Serve(nil, h)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func DoorHandler(w http.ResponseWriter, r *http.Request) {
	arr := strings.Split(r.URL.Path[1:], "/")
	if len(arr) > 3 {
		state, err := strconv.ParseInt(arr[1], 10, 16)
		if err != nil {
			http.Error(w, "invalid parameter", 400)
			return
		}
		UserCnt, err := strconv.ParseInt(arr[2], 10, 16)
		if err != nil {
			http.Error(w, "invalid parameter", 400)
			return
		}
		if err == nil {
			var d Door
			db.FirstOrCreate(&d)
			curState := state > 0
			if d.Open != curState {
				d.LastDoorChange = time.Now()
			}
			d.Open = curState
			d.UserCount = int(UserCnt)
			d.UserNames = sql.NullString{String: arr[3], Valid: true}
			db.Save(&d)
			if d.Open {
				fmt.Fprintf(w, "now Open")
			} else {
				fmt.Fprintf(w, "now Closed")
			}
			return
		}
	}
	fmt.Fprintf(w, "NOK")
}

func SensorHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf(r.URL.Path)
	if r.Method == "PUT" {
		HandleSensorJSON(w, r)
		return
	}

	arr := strings.Split(r.URL.Path[1:], "/")[1:]
	if len(arr) >= 3 {
		f, err := strconv.ParseFloat(arr[2], 32)
		if err != nil {
			fmt.Fprintf(w, "NOK\nparsefloat: %s", err.Error())
			return
		}
		switch arr[0] {
		case "beverage_supply":
			if len(arr) != 5 {
				fmt.Fprintf(w, "NOK\nlength wrong! want %d got %d", 5, len(arr))
				return
			}
			err = UpdateOrInsertSensorByName(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f),
				Unit: arr[3], Name: arr[4]}, true)

		default:
			switch len(arr) {
			case 3:
				err = UpdateOrInsertSensor(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f)})
			case 4:
				err = UpdateOrInsertSensor(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f),
					Unit: arr[3]})
			case 5:
				err = UpdateOrInsertSensor(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f),
					Unit: arr[3], Description: arr[4]})
			default:
				fmt.Fprintf(w, "NOK\nlength wrong! want 3/4/5, got %s", len(arr))
				return
			}
		}
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Fail: %s", err)
		} else {
			fmt.Fprintf(w, "Ok")
		}
		return
	}
	fmt.Fprintf(w, "NOK\nlength wrong! want >=3 got %d", len(arr))
}

func HandleSensorJSON(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "NOK\nreading request body: "+err.Error(), 400)
		return
	}
	var updates []UpdateParam
	err = json.Unmarshal(b, &updates)
	if err != nil {
		http.Error(w, "NOK\ndecoding JSON: "+err.Error(), 400)
		return
	}
	errs := []error{}
	for _, up := range updates {
		err = UpdateOrInsertSensor(up)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%+v", errs)
	} else {
		fmt.Fprintf(w, "Ok")
	}
}

func SpaceapiHandler(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	headers.Add("Content-Type", "application/json; charset=utf-8")
	headers.Add("Cache-Control", "no-cache")
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

	doorState := GetDoorState()
	state["lastchange"] = doorState.LastDoorChange.Unix()
	state["open"] = doorState.Open
	m["open"] = doorState.Open
	outSensors := make(map[string]interface{})

	types := GetSensorTypes()
	for i := range types {
		s := GetSensorsByType(types[i])
		sensorType := make([]map[string]interface{}, len(s))
		for j := range s {
			curSensor := make(map[string]interface{})
			curSensor["location"] = s[j].Location
			curSensor["value"] = s[j].Value
			curSensor["ext_modified"] = s[j].UpdatedAt
			if s[j].Description.Valid {
				curSensor["description"] = s[j].Description.String
			}
			if s[j].Unit.Valid {
				curSensor["unit"] = s[j].Unit.String
			}
			curSensor["name"] = s[j].Name
			sensorType[j] = curSensor
		}
		outSensors[types[i]] = sensorType
	}
	sensorType := make([]map[string]interface{}, 1)
	curSensor := make(map[string]interface{})
	curSensor["value"] = doorState.UserCount
	if doorState.UserNames.Valid {
		curSensor["names"] = doorState.UserNames.String
	} else {
		curSensor["names"] = ""
	}
	sensorType[0] = curSensor
	outSensors["people_now_present"] = sensorType

	state["sensors"] = outSensors
	bytes, err := json.MarshalIndent(f, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	_, err = w.Write(bytes)
	if err != nil {
		log.Fatal(err)
	}
}

func GetSensorTypes() []string {
	var arr []struct{ Type string }
	db.Raw("SELECT DISTINCT type FROM sensors").Scan(&arr)
	if db.Error != nil {
		log.Fatal("getting sensor types", db.Error)
	}
	ret := []string{}
	for _, s := range arr {
		ret = append(ret, s.Type)
	}
	return ret
}

func GetSensorsByType(id string) (arr []Sensor) {
	db.Find(&arr, Sensor{Type: id})
	return
}

func InitDatabase(f *os.File) {
	db, err = gorm.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	//db.LogMode(true)
	db.SetLogger(gorm.Logger{log.New(f, "\r\n", 0)})
	db.AutoMigrate(&Sensor{})
	db.AutoMigrate(&Door{})
}

func GetDoorState() Door {
	var d Door
	db.First(&d)
	if db.Error != nil {
		log.Fatal(db.Error)
		return Door{Open: false, UserCount: 0, UserNames: sql.NullString{String: "", Valid: true}}
	}
	return d
}

func UpdateOrInsertSensor(p UpdateParam) error {
	return UpdateOrInsertSensorByName(p, false)
}

func UpdateOrInsertSensorByName(p UpdateParam, ByName bool) error {
	var sensor Sensor
	var name string

	if ByName && p.Name != "" {
		name = p.Name
	} else {
		name = p.SensorType + "_" + p.Location
	}
	db.FirstOrCreate(&sensor, Sensor{Type: p.SensorType, Location: p.Location, Name: name})
	sensor.Location = p.Location

	if len(p.Unit) > 0 {
		sensor.Unit = sql.NullString{p.Unit, true}
	}
	if len(p.Description) > 0 {
		sensor.Description = sql.NullString{p.Description, true}
	}
	sensor.Value = p.Value
	db.Save(&sensor)
	return nil
}
