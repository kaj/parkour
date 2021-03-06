package parkour

import (
	"gopkg.in/mgo.v2/bson"
	"time"
    "fmt"
)

// Collection entry for database
type Bout struct {
	Id     bson.ObjectId `bson:"_id"`
	User   string
	Other  string
	Course string
	Lab    string
	Logs   []LogEntry
}

type LogEntry struct {
	Timestamp time.Time
	Entry     string // Enum? "self", "other", "pause"
	Duration  int
}

func (bout *Bout) With() User {
    return GetUser(bout.Other)
}
func (bout *Bout) RealName1() string {
    return GetUser(bout.User).Name
}
func (bout *Bout) OtherUser(me User) User {
    fmt.Println("In Othername;", me.Kthid, bout.User, bout.Other)
    if me.Kthid == bout.User {
        return GetUser(bout.Other)
    } else {
        return GetUser(bout.User)
    }
}

func (bout *Bout) CourseLab() string {
    return fmt.Sprintf("%s %s", courses[bout.Course], labs[bout.Lab])
}
func (bout *Bout) LabName() string {
    return labs[bout.Lab]
}

func (bout *Bout) Starttime() string {
    if len(bout.Logs) > 0 {
        return bout.Logs[0].Timestamp.Format("2006-01-02 15:04")
    } else {
        return ""
    }
}

func (bout *Bout) Duration() string {
    logs := bout.Logs
    if len(logs) > 0 {
        return formatSeconds(int(logs[len(logs)-1].Timestamp.Sub(logs[0].Timestamp).Seconds()))
    } else {
        return ""
    }
}

func (bout *Bout) GetLogs() []LogEntry {
    for i := range(bout.Logs) {
        if i > 0 {
            bout.Logs[i-1].Duration =
                int(bout.Logs[i].Timestamp.Sub(bout.Logs[i-1].Timestamp).Seconds())
        }
    }
    return bout.Logs
}

func (bout *Bout) DriverDurations(prim User) *Balance {
    my, others := 0, 0
    for _, log := range bout.GetLogs() {
        if log.Entry == bout.User {
            my += log.Duration
        } else if log.Entry == bout.Other {
            others += log.Duration
        }
    }
    if prim.Kthid == bout.User {
        return &Balance{bout.User, my, bout.Other, others}
    } else {
        return &Balance{bout.Other, others, bout.User, my}
    }
}

func (log *LogEntry) What(user User) string {
    if log.Entry == "pause" {
        return "Pause"
    } else if log.Entry == user.Kthid {
        return "Driver"
    } else {
        return "Navigator"
    }
}
func (log *LogEntry) DriverName() string {
    if log.Entry == "pause" {
        return "Pause"
    } else {
        return GetUser(log.Entry).Firstname + " is driver"
    }
}

func (log *LogEntry) FormatDuration() string {
    return formatSeconds(log.Duration)
}

func formatSeconds(seconds int) string {
    if seconds < 60 {
        return fmt.Sprintf("%d seconds", seconds)
    }
    return fmt.Sprintf("%d minutes %d seconds", seconds/60, seconds%60)
}
