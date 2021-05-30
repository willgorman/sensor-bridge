#!/bin/bash

systemctl stop sensor-bridge
systemctl disable sensor-bridge

rm -rf /etc/sensor-bridge
