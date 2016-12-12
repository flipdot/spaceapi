package main

import (
    "fmt"
    "flag"
    "net/http"
    "net/http/fcgi"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "log"
    "strings"
    "strconv"
    "encoding/json"
    "io/ioutil"
	"os"
)

type SensorTypes struct{
    id int
    name string
}

type Sensor struct{
    location string
    value float32
    desc string
    unit string
}

var db *sql.DB
var err error
var local = flag.String("local", "", "serve as webserver, example: 0.0.0.0:8000")

func main() {
	f, err := os.OpenFile("spaceapi.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
    log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(f)
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
    for i := range types{
        s := getSensorsByType(types[i].id)
        sensorType := make([]map[string]interface{},len(s))
        //sensorType := make([]Sensor,len(s))
        for j := range s{
            curSensor := make(map[string]interface{})
            curSensor["location"] = s[j].location
            curSensor["value"] = s[j].value
            if len(s[j].desc) > 0 {
                curSensor["description"] = s[j].desc
            }
            if len(s[j].unit) > 0 {
                curSensor["unit"] = s[j].unit
            }
            sensorType[j] = curSensor
            //sensorType[j] = s[j]
        }
        sensor[types[i].name] = sensorType
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
    rows, err := db.Query("select id, name from sensortypes")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    var arr []SensorTypes
    for rows.Next() {
        var id int
        var name string
        err = rows.Scan(&id, &name)
        if err != nil {
            log.Fatal(err)
        }
        var d SensorTypes
        d.id = id
        d.name = name
        arr = append(arr, d)
    }
    return arr
}

func getSensorsByType(id int) []Sensor{
    stmt, err := db.Prepare("select location, value, unit, description from sensors where sensortype = ?")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()
    rows, err := stmt.Query(id)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    var arr []Sensor
    for rows.Next() {
        var s Sensor
        var desc sql.NullString
        var unit sql.NullString
        err = rows.Scan(&s.location, &s.value, &unit, &desc)
        if err != nil {
            log.Fatal(err)
        }
        if unit.Valid {
            s.unit = unit.String
        }
        if desc.Valid {
            s.desc = desc.String
        }
        arr = append(arr, s)
    }
    return arr
}


func initDatabase(){

    db, err = sql.Open("sqlite3", "./foo.db")
    if err != nil {
        log.Fatal(err)
    }

    sqlStmt := `create table IF NOT EXISTS sensortypes (id integer not null primary key, name text);
    create table IF NOT EXISTS sensors (id integer not null primary key,
        sensortype integer,
        location text,
        value real,
        unit text,
        description text,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(sensortype) REFERENCES sensortypes(id)
    );`

    _, err = db.Exec(sqlStmt)
    if err != nil {
        log.Printf("%q: %s\n", err, sqlStmt)
        return
    }
}

func GetLastUpdate() int{
    rows := db.QueryRow("select strftime('%s', timestamp) from sensors ORDER BY timestamp DESC limit 1;")
    var ts int
    err = rows.Scan(&ts)
    if err != nil {
		return 0
    }
    return ts
}


func updateOrInsertValue(sensortype string, location string, value float32, unit string, desc string){
    stmt, err := db.Prepare("select id from sensortypes where name = ?")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()

    var id int;
    err = stmt.QueryRow(sensortype).Scan(&id)
    stmt.Close()
    if err != nil {
        id = insertType(sensortype)
    }
    stmt, err = db.Prepare("select id from sensors where location = ? and sensortype = ?")
    if err != nil {
        log.Fatal(err)
    }
    err = stmt.QueryRow(location, id).Scan(&id)
    if err != nil {
        insertNewSensor(id, location, value, unit, desc)
        return
    }
    if len(desc) > 0 {
        updateSensorUnitDesc(id, value, unit, desc)
    } else if len(unit) > 0 {
        updateSensorUnit(id, value, unit)
    } else {
        updateSensor(id, value)
    }
}

func updateSensorUnitDesc(id int, value float32, unit string, desc string) {
    tx, err := db.Begin()
    stmtInsert, err := tx.Prepare("UPDATE sensors SET value = ?, unit = ?, description = ?, timestamp=datetime('now') WHERE id = ?")
    if err != nil {
        log.Fatal(err)
    }
    defer stmtInsert.Close()
    _, err = stmtInsert.Exec(value, unit, desc, id)
    if(err != nil) {
        log.Fatal(err)
    }
    tx.Commit()
}

func updateSensorUnit(id int, value float32, unit string) {
    tx, err := db.Begin()
    stmtInsert, err := tx.Prepare("UPDATE sensors SET value = ?, unit = ?, timestamp=datetime('now') WHERE id = ?")
    if err != nil {
        log.Fatal(err)
    }
    defer stmtInsert.Close()
    _, err = stmtInsert.Exec(value, unit, id)
    if(err != nil) {
        log.Fatal(err)
    }
    tx.Commit()
}

func updateSensor(id int, value float32) {
    tx, err := db.Begin()
    stmtInsert, err := tx.Prepare("update sensors set value = ?, timestamp=datetime('now') WHERE id = ?")
    if err != nil {
        log.Fatal(err)
    }
    defer stmtInsert.Close()
    _, err = stmtInsert.Exec(value, id)
    if(err != nil) {
        log.Fatal(err)
    }
    tx.Commit()
}

func insertNewSensor(id int, location string, value float32, unit string, desc string) {
    tx, err := db.Begin()
    stmtInsert, err := tx.Prepare("INSERT INTO sensors(sensortype, location, value, unit, description) VALUES(?,?,?,?,?)")
    if err != nil {
        log.Fatal(err)
    }
    defer stmtInsert.Close()
    _, err = stmtInsert.Exec(id, location, value, unit, desc)
    if(err != nil) {
        log.Fatal(err)
    }
    tx.Commit()
}


func insertType(name string) int {
    tx, err := db.Begin()
    if err != nil {
        log.Fatal(err)
    }
    stmt, err := tx.Prepare("insert into sensortypes(name) values(?)")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()
    _, err = stmt.Exec(name)
    tx.Commit()

    stmt, err = db.Prepare("select id from sensortypes where name = ?")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()

    var id int;
    err = stmt.QueryRow(name).Scan(&id)
    stmt.Close()
    if err != nil {
        log.Fatal(err)
    }
    return id
}
