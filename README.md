# LORO
### LORO Only Repeats Output
![](https://media.giphy.com/media/5PSPV1ucLX31u/giphy-downsized.gif)

### What?
LORO is a CLI to query AWS CloudWatch-Logs groups, streams and events.

### Why?
Sometimes you just want to tail some logs.

#### Inspiration/Sources
- https://github.com/segmentio/cwlogs
- https://github.com/jorgebastida/awslogs
- https://github.com/mmcquillan/lawsg

#### Notes
This repository contains a big chunk of code that is similar to segmentio/cwlogs, you might say "fork/PR that repo".
I believe the intentions of segmentio/cwlogs and the ones of this repo are different, as segmentio/cwlogs supports segements's CWLogs format, and this repo is intended to be more generic.
Hence it made sense to separate the repository, but I still would like to credit segmentio/cwlogs and the Segment team for their work.
