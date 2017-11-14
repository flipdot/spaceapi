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

type SensorTypes struct {
	gorm.Model
	Name string `sql:"size:255;unique;index"`
}

type Sensor struct {
	gorm.Model
	Type        SensorTypes `gorm:"ForeignKey:TypeRefer"`
	TypeRefer   int
	Location    string
	Value       float32
	Description sql.NullString
	Unit        sql.NullString
	Name        sql.NullString
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
	InitDatabase()
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
			fmt.Fprintf(w, "NOK")
			return
		}
		switch arr[0] {
		case "beverage_supply":
			if len(arr) != 5 {
				fmt.Fprintf(w, "NOK")
				return
			}
			UpdateOrInsertSensorByName(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f),
				Unit: arr[3], Name: arr[4]}, true)

		default:
			switch len(arr) {
			case 3:
				UpdateOrInsertSensor(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f)})
			case 4:
				UpdateOrInsertSensor(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f),
					Unit: arr[3]})
			case 5:
				UpdateOrInsertSensor(UpdateParam{SensorType: arr[0], Location: arr[1], Value: float32(f),
					Unit: arr[3], Description: arr[4]})
			default:
				fmt.Fprintf(w, "NOK")
				return
			}
		}
		fmt.Fprintf(w, "Ok")
		return

	}
	fmt.Fprintf(w, "NOK")
}

func HandleSensorJSON(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "NOK\n"+err.Error(), 400)
		return
	}
	var updates []UpdateParam
	err = json.Unmarshal(b, &updates)
	if err != nil {
		http.Error(w, "NOK\n"+err.Error(), 400)
		return
	}
	for _, up := range updates {
		UpdateOrInsertSensor(up)
	}
	fmt.Fprintf(w, "Ok")
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
	sensor := make(map[string]interface{})

	types := GetSensorTypes()
	for i := range types {
		s := GetSensorsByType(types[i].Model.ID)
		sensorType := make([]map[string]interface{}, len(s))
		//sensorType := make([]Sensor,len(s))
		for j := range s {
			curSensor := make(map[string]interface{})
			curSensor["location"] = s[j].Location
			curSensor["value"] = s[j].Value
			if s[j].Description.Valid {
				curSensor["description"] = s[j].Description.String
			}
			if s[j].Unit.Valid {
				curSensor["unit"] = s[j].Unit.String
			}
			if s[j].Name.Valid {
				curSensor["name"] = s[j].Name.String
			}
			sensorType[j] = curSensor
			//sensorType[j] = s[j]
		}
		sensor[types[i].Name] = sensorType
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
	sensor["people_now_present"] = sensorType

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

func GetSensorTypes() []SensorTypes {
	var arr []SensorTypes
	db.Find(&arr)
	if db.Error != nil {
		log.Fatal("bla err")
	}
	return arr
}

func GetSensorsByType(id uint) []Sensor {
	var arr []Sensor
	db.Find(&arr, Sensor{TypeRefer: int(id)})
	return arr
}

func InitDatabase() {
	db, err = gorm.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&SensorTypes{})
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

func UpdateOrInsertSensor(p UpdateParam) {
	UpdateOrInsertSensorByName(p, false)
}

func UpdateOrInsertSensorByName(p UpdateParam, ByName bool) {
	var sType SensorTypes
	var sensor Sensor

	db.FirstOrCreate(&sType, SensorTypes{Name: p.SensorType})
	if ByName {
		db.FirstOrCreate(&sensor, Sensor{TypeRefer: int(sType.ID),
			Name: sql.NullString{p.Name, true}})
	} else {
		db.FirstOrCreate(&sensor, Sensor{TypeRefer: int(sType.ID), Location: p.Location})
	}
	sensor.Location = p.Location

	if len(p.Name) > 0 {
		sensor.Name = sql.NullString{p.Name, true}
	}

	if len(p.Unit) > 0 {
		sensor.Unit = sql.NullString{p.Unit, true}
	}
	if len(p.Description) > 0 {
		sensor.Description = sql.NullString{p.Description, true}
	}
	sensor.Value = p.Value
	db.Save(&sensor)
}
