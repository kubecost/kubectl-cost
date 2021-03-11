# Presentation script

Commands to run for presentation recording of usage.

## Regular usage 

```
asciinema rec -c '/bin/bash -c "
    clear;
    echo \$ kubectl cost namespace;
    kubectl cost namespace;
    sleep 4; clear;
    echo \$ kubectl cost deployment --show-cpu;
    kubectl cost deployment --show-cpu;
    sleep 4; clear;
    echo \$ kubectl cost controller --show-pv;
    kubectl cost controller --show-pv;
    sleep 4; clear;
    echo \$ kubectl cost label --historical --window 7d -l app;
    kubectl cost label --historical --window 7d -l app;
    sleep 6"' \
    regular.cast
    
docker run --rm -v $PWD:/data asciinema/asciicast2gif regular.cast regular.gif
```


## TUI usage

```
asciinema rec -c '/bin/bash -c "
    clear;
    echo \$ kubectl cost tui;
    sleep 1;
    kubectl cost tui"' tui.cast
```

```
m
p
-- wait --
ESC
ENTER
DOWN
ENTER
-- wait --
ENTER
DOWN
ENTER
-- wait --
ESC
ENTER
DOWN
ENTER
-- wait --
ESC
p
c
-- wait --
CTRL-C
```

```
docker run --rm -v $PWD:/data asciinema/asciicast2gif tui.cast tui.gif
```
