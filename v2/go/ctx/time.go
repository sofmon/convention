package ctx

import "time"

/*
	Time
*/

func (ctx Context) Now() time.Time {

	if !ctx.IsProdEnv() {

		r := ctx.Request()

		if r != nil {

			nowStr := r.Header.Get(HTTPHeaderTimeNow)

			if nowStr != "" {

				now, err := time.Parse(time.RFC3339, nowStr)
				if err != nil {
					ctx.LogWarnf("failed to parse %s header: %s", HTTPHeaderTimeNow, err.Error())
					return time.Now().UTC()
				}

				return now.UTC()
			}

		}

	}

	return time.Now().UTC()
}
