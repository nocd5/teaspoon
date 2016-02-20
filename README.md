# teaspoon
Terminal Serial Plotter

## List COM Ports

```sh
tsp -l
```

![list ports](ports.png)

## Connect

```sh
tsp -p COM6 -b 9600
```

![connect](connect.png)

A Delimiter option `--delimiter="\"\r\n\""` may be needed if you use Serial.println in Arduino.  
Because Serial.println terminate string by **CRLF**

## Run

![run](run.png)

## Quit

Press `Q` key to quit
