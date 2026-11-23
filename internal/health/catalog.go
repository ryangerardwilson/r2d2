package health

import "time"

type kind int

const (
	service   kind = iota // long-running unit, must be active and running
	timer                 // timer plus the last run of its oneshot
	boot                  // boot oneshot, judged by its last result
	fresh                 // object a timer writes, judged by its mtime
	present               // object written on change, judged by existence
	skippable             // object written on change, skipped while its source is empty
)

const day = 24 * time.Hour

type check struct {
	kind   kind
	name   string // unit name; objects use their s3 URI
	unit   string // service a timer starts
	bucket string
	key    string
	every  time.Duration
	why    string // reason a skippable object is waiting
}

// catalog is read from the user units and the publish code of each app.
// Omarchy and desktop units are not here. The daily sql dump stays on the
// host, so the last result of nebula-sql-backup.service stands for it.
var catalog = []check{
	{kind: service, name: "tunnel.service"},
	{kind: service, name: "looptab.service"},
	{kind: service, name: "antibloat-observer.service"},
	{kind: timer, name: "gmail.timer", unit: "gmail.service", every: time.Hour},
	{kind: timer, name: "gdrive.timer", unit: "gdrive.service", every: time.Hour},
	{kind: timer, name: "nebula-keep.timer", unit: "nebula-keep.service", every: 10 * time.Minute},
	{kind: timer, name: "nebula-sql-backup.timer", unit: "nebula-sql-backup.service", every: day},
	{kind: timer, name: "tesseract-snapshot.timer", unit: "tesseract-snapshot.service", every: 15 * time.Minute},
	{kind: timer, name: "rgw-slack-quickshell-plugin.timer", unit: "rgw-slack-quickshell-plugin.service", every: 30 * time.Minute},
	{kind: timer, name: "genesys.timer", unit: "genesys.service", every: day},
	{kind: boot, name: "tesseract-restore.service"},
	{kind: fresh, bucket: "backups", key: "c3po.sqlite", every: 10 * time.Minute},
	{kind: fresh, bucket: "backups", key: "tesseract.sqlite", every: 15 * time.Minute},
	{kind: fresh, bucket: "backups", key: "rgw_slack_quickshell_plugin.sqlite", every: 30 * time.Minute},
	{kind: present, bucket: "backups", key: "musica.sqlite"},
	{kind: present, bucket: "backups", key: "antibloat.sqlite"},
	{kind: skippable, bucket: "backups", key: "crusade.sqlite", why: "crusade skips the put while its task table is empty"},
}

func (c check) title() string {
	if c.name != "" {
		return c.name
	}
	return "s3://" + c.bucket + "/" + c.key
}
