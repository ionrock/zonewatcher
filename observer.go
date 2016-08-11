package zonewatcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

const (
	ZONE_CREATED    = "ZONE_CREATE"
	ZONE_DELETED    = "ZONE_DELETE"
	STATUS_CANCELED = "CANCELED"
	STATUS_FINISHED = "FINISHED"
	STATUS_WAITING  = "WAITING"
)

func Exists(zone string, ns string) string {
	c := []string{"dig", "+short"}

	if strings.Contains(ns, ":") {
		parts := strings.Split(ns, ":")
		c = append(c, "-p", parts[1], fmt.Sprintf("@%s", parts[0]))
	}
	c = append(c, zone, "SOA")

	log.Print(strings.Join(c, " "))

	cmd := exec.Command(c[0], c[1:]...)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	log.Printf("output: %s", bytes.Trim(out, "\n"))

	if len(bytes.Trim(out, "\n")) == 0 {
		return ZONE_DELETED
	}
	return ZONE_CREATED
}

func timestamp() int64 {
	return time.Now().UTC().Unix()
}

type Observer struct {
	Zone     string `json:"zone"`
	Ns       string `json:"ns"`
	Start    int64  `json:"start"`
	Finish   int64  `json:"finish"`
	Duration int64  `json:"duration"`
	Interval int64  `json:"interval"`
	Timeout  int64  `json:"timeout"`
	State    string `json:"state"`
	Status   string `json:"status"`
	exit     bool
}

func (o *Observer) Name() string {
	return fmt.Sprintf("%s-%s-%s", o.Zone, o.Ns, o.State)
}

func (o *Observer) Watch(db *bolt.DB) {
	o.Open(db)

	// close as cancelled by default
	defer o.Close(db, STATUS_CANCELED)
	o.Status = STATUS_WAITING

	if o.Timeout == 0 {
		o.Timeout = 15
	}

	if o.Interval == 0 {
		o.Interval = 1
	}

	timeout := time.NewTimer(time.Duration(o.Timeout) * time.Second).C

	for {
		select {
		case <-timeout:
			o.Stop()
		default:
			if o.exit {
				log.Printf("Exiting %s", o.Name())
				o.Close(db, STATUS_CANCELED)
				return
			}
			if Exists(o.Zone, o.Ns) == o.State {
				o.Close(db, STATUS_FINISHED)
				return
			}
			time.Sleep(time.Duration(o.Interval) * time.Second)
		}
	}
}

func (o *Observer) Stop() {
	o.exit = true
}

func (o *Observer) Close(db *bolt.DB, status string) {
	o.Finish = timestamp()
	o.Duration = o.Finish - o.Start
	o.Status = status
	log.Printf("Finished %s %d %d %s", o.Name(), o.Finish, o.Duration, o.State)
}

func (o *Observer) Open(db *bolt.DB) {
	o.Start = timestamp()
	o.Status = STATUS_WAITING
	log.Printf("Started %s %d", o.Name(), o.Start)
}

func StopObservers(observers []Observer) {
	for _, o := range observers {
		log.Print("Stopping: ", o)
		o.Stop()
		log.Print("Stopped: ", o)
	}
}

func (o *Observer) Sync(db *bolt.DB) {
	bucket := o.Name()
	key := strconv.FormatInt(timestamp(), 10)
	val := []byte{}
	buf := bytes.NewBuffer(val)
	err := json.NewEncoder(buf).Encode(o)

	if err != nil {
		panic(err)
	}

	contentType := "application/json"

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		if err = b.Put([]byte(key), val); err != nil {
			return err
		}
		return b.Put([]byte(fmt.Sprintf("%s-ContentType", key)), []byte(contentType))
	})
}
