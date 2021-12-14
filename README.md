# loro

## Loro Only Repeats Output

![loro](https://media.giphy.com/media/5PSPV1ucLX31u/giphy-downsized.gif)

## What?

**loro** is a CLI to query AWS CloudWatch-Logs groups, streams and events.

## Why?

Sometimes you just want to tail some logs.

## How?

### Get logs

Get logs for a streamgroup:

```
loro get /streamgroup/
```

Select a single stream:

```
loro get /streamgroup/ -p stream
```

Print raw logs:

```
loro get -r /streamgroup/
```

Tail a log:

```
loro get -f /streamgroup/
```

### Find streams or groups

List streams

```
loro list streams /streamgroup/
```

List groups:

```
loro list groups /streamgroup/partialname
```

### Get help

All commands contain help documentation by using `--help` flag

```bash
> loro get --help
Get logs from a group or stream

Usage:
  loro get [flags]

Flags:
  -f, --follow            Follow log streams
  -o, --format string     Format template for displaying log events (default "[ {{ uniquecolor (print .Stream) }} ] {{ .TimeShort }} - {{ .Event.message }}")
  -h, --help              help for get
  -m, --max-streams int   Maximum number of streams to fetch from (for prefix search) (default 10)
  -p, --prefix string     Stream Name or prefix
  -r, --raw               Raw JSON output
  -s, --since string      Fetch logs since timestamp (e.g. 2013-01-02T13:23:37), relative (e.g. 42m for 42 minutes), or all for all logs (default "1h")
  -u, --until string      Fetch logs until timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes) (default "now")

Global Flags:
      --config string   config file (default is $HOME/.loro.yaml)
```

#### Inspiration and Sources

- https://github.com/segmentio/cwlogs
- https://github.com/jorgebastida/awslogs
- https://github.com/mmcquillan/lawsg

#### Notes

Why create a new piece of software?

- `segmentio/cwlogs` Is great, not only a big inspiration for LORO but also a big part of the lib codebase was forked from them. But, its built for supporting segment.io log format and while that makes sense and works great, its not generic enought for many other cases or supporting other log formats
- `jorgebastida/awslogs` I believe binary distribution is better for this tool and golang is a more powerfull and faster language when dealing with many logs

Hence it made sense to separate the repository, but I still would like to credit again `segmentio/cwlogs` and the Segment team for their work as it provides the base for this software.
