# Tool for decoding GNSS systems

What is working:
- GPS (acq + track + decode + pvt)
- Galileo (acq + track, decode is failing on CRC on our recordings, added pvt solver capabilities)
- Glonass (zero real recordings, using sim (sim have problem with eph) we can see that acq + track + decode is working)
