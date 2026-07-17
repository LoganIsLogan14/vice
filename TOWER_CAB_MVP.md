# Tower Cab MVP

This branch adds an initial top-down Tower Cab display.

## What it does

- Adds a new `tower` package and `TowerCabPane`.
- Adds a plane button to the main toolbar to switch between the normal radar display and Tower Cab.
- Centers the display on the scenario's primary airport.
- Draws every current VICE track as a heading-oriented top-down triangle with a callsign label.
- Supports mouse-wheel zoom from 1 to 40 NM.

## Current intentional limitations

- VICE still deletes arrivals before landing.
- Departures still appear after takeoff.
- No ground movement is simulated.
- Airport geometry is represented only by a center crosshair.
- No panning, aircraft sprites, taxiways, runways, or ASDE-X symbology yet.

## Testing

1. Build VICE using its required Go toolchain (currently Go 1.25 or newer).
2. Start any scenario.
3. Click the plane icon in the main toolbar.
4. Use the mouse wheel to zoom.
5. Click the plane icon again to return to STARS/ERAM.
