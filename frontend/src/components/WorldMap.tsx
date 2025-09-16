import React, { useState, useEffect, useRef, useImperativeHandle, forwardRef } from 'react';
import { MapContainer, Marker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { Telemetry } from '../types';
import { FormControl, InputLabel, Select, MenuItem, Box, SelectChangeEvent } from '@mui/material';

// Fix for default Leaflet icons
delete (L.Icon.Default.prototype as any)._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
  iconUrl: require('leaflet/dist/images/marker-icon.png'),
  shadowUrl: require('leaflet/dist/images/marker-shadow.png'),
});

interface WorldMapProps {
  telemetry: Telemetry[];
  onNavigateToVehicleDetail?: (vehicleId: string) => void;
}

export interface WorldMapRef {
  focusOnVehicle: (vehicleId: string) => void;
}

// Map style configurations
interface MapStyle {
  id: string;
  name: string;
  url: string;
  attribution: string;
}

const MAP_STYLES: MapStyle[] = [
  {
    id: 'osm',
    name: 'OpenStreetMap Standard',
    url: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
    attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
  },
  {
    id: 'cartodb-light',
    name: 'CartoDB Positron (Light)',
    url: 'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png',
    attribution: '© <a href="https://carto.com/attributions">CartoDB</a>'
  },
  {
    id: 'cartodb-dark',
    name: 'CartoDB Dark Matter',
    url: 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png',
    attribution: '© <a href="https://carto.com/attributions">CartoDB</a>'
  },
  {
    id: 'stamen-terrain',
    name: 'Stamen Terrain',
    url: 'https://stamen-tiles.a.ssl.fastly.net/terrain/{z}/{x}/{y}.jpg',
    attribution: '© <a href="http://stamen.com">Stamen</a>'
  },
  {
    id: 'esri-satellite',
    name: 'Esri World Imagery (Satellite)',
    url: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
    attribution: '© <a href="https://www.esri.com/">Esri</a>'
  },
  {
    id: 'carto-voyager',
    name: 'Simplified (Colorful Kid Map)',
    url: 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png',
    attribution: '© <a href="https://carto.com/">CARTO</a>'
  },
  {
    id: 'carto-minimal',
    name: 'Minimal (No Roads or Labels)',
    url: 'https://{s}.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}{r}.png',
    attribution: '© <a href="https://carto.com/">CARTO</a>'
  }
];

// Component to handle dynamic tile layer switching
const DynamicTileLayer: React.FC<{ selectedStyle: MapStyle }> = ({ selectedStyle }) => {
  const map = useMap();

  useEffect(() => {
    // Remove all existing tile layers
    map.eachLayer((layer) => {
      if (layer instanceof L.TileLayer) {
        map.removeLayer(layer);
      }
    });

    // Add the new tile layer
    const newTileLayer = L.tileLayer(selectedStyle.url, {
      attribution: selectedStyle.attribution,
      maxZoom: 19,
    });
    
    newTileLayer.addTo(map);

    return () => {
      // Cleanup on unmount
      map.removeLayer(newTileLayer);
    };
  }, [map, selectedStyle]);

  return null;
};

// Component to capture map instance
const MapInstanceCapture: React.FC<{ mapRef: React.MutableRefObject<L.Map | null> }> = ({ mapRef }) => {
  const map = useMap();
  
  useEffect(() => {
    mapRef.current = map;
  }, [map, mapRef]);

  return null;
};

// Custom Marker component that stores its reference
interface VehicleMarkerProps {
  vehicle: Telemetry;
  icon: L.Icon;
  markersRef: React.MutableRefObject<Map<string, L.Marker>>;
  children: React.ReactNode;
}

const VehicleMarker: React.FC<VehicleMarkerProps> = ({ vehicle, icon, markersRef, children }) => {
  return (
    <Marker
      position={[vehicle.location.lat, vehicle.location.lon]}
      icon={icon}
      ref={(marker) => {
        if (marker) {
          markersRef.current.set(vehicle.vehicle_id, marker);
        } else {
          markersRef.current.delete(vehicle.vehicle_id);
        }
      }}
    >
      {children}
    </Marker>
  );
};

const WorldMap = forwardRef<WorldMapRef, WorldMapProps>(({ telemetry, onNavigateToVehicleDetail }, ref) => {
  // Map instance reference
  const mapRef = useRef<L.Map | null>(null);
  // Store marker references for popup control
  const markersRef = useRef<Map<string, L.Marker>>(new Map());
  // State for selected map style with localStorage support
  const [selectedStyleId, setSelectedStyleId] = useState<string>(() => {
    const savedStyle = localStorage.getItem('worldmap-style');
    return savedStyle || 'osm'; // Default to OpenStreetMap
  });

  // Get the selected style object
  const selectedStyle = MAP_STYLES.find(style => style.id === selectedStyleId) || MAP_STYLES[0];

  // Formatting helpers for popup values
  const format1 = (n: number | undefined | null): string => {
    const v = Number(n);
    if (!isFinite(v)) return '0.0';
    return (Math.round(v * 10) / 10).toFixed(1);
  };

  const format0 = (n: number | undefined | null): string => {
    const v = Number(n);
    if (!isFinite(v)) return '0';
    return String(Math.round(v));
  };

  // Handle style change
  const handleStyleChange = (event: SelectChangeEvent<string>) => {
    const newStyleId = event.target.value;
    setSelectedStyleId(newStyleId);
    localStorage.setItem('worldmap-style', newStyleId);
  };

  // Create custom icons for EV and ICE vehicles
  const evIcon = new L.Icon({
    iconUrl: '/ev-vehicle-icon.svg',
    iconSize: [32, 32],
    iconAnchor: [16, 28],
    popupAnchor: [0, -28],
    className: 'custom-div-icon',
  });

  const iceIcon = new L.Icon({
    iconUrl: '/ice-vehicle-icon.svg',
    iconSize: [32, 32],
    iconAnchor: [16, 28],
    popupAnchor: [0, -28],
    className: 'custom-div-icon',
  });

  const chargingStationIcon = new L.Icon({
    iconUrl: '/charging-station-icon.svg',
    iconSize: [32, 32],
    iconAnchor: [16, 30],
    popupAnchor: [0, -30],
    className: 'custom-div-icon',
  });

  // Sample vehicle data if no telemetry provided
  const sampleVehicles: Telemetry[] = [
    {
      vehicle_id: 'sample-ev-1',
      location: { lat: 51.5, lon: -0.09 }, // London
      type: 'EV',
      speed: 45,
      battery_level: 85,
      emissions: 0,
      status: 'active',
      timestamp: new Date().toISOString(),
    },
    {
      vehicle_id: 'sample-ice-1',
      location: { lat: 40.7128, lon: -74.0060 }, // New York
      type: 'ICE',
      speed: 32,
      fuel_level: 65,
      emissions: 120,
      status: 'active',
      timestamp: new Date().toISOString(),
    },
    {
      vehicle_id: 'sample-ev-2',
      location: { lat: 35.6762, lon: 139.6503 }, // Tokyo
      type: 'EV',
      speed: 28,
      battery_level: 92,
      emissions: 0,
      status: 'active',
      timestamp: new Date().toISOString(),
    },
  ];

  // Use provided telemetry or sample data
  const vehicles: Telemetry[] = telemetry && telemetry.length > 0 ? telemetry : sampleVehicles;

  // Filter vehicles with valid location data
  const validVehicles = vehicles.filter((vehicle: Telemetry) => 
    vehicle.location && 
    typeof vehicle.location.lat === 'number' && 
    typeof vehicle.location.lon === 'number'
  );

  const getVehicleType = (vehicle: Telemetry): 'ICE' | 'EV' => {
    if (vehicle.type === 'EV' || vehicle.type === 'ICE') {
      return vehicle.type;
    }
    return vehicle.battery_level !== undefined && vehicle.battery_level !== null ? 'EV' : 'ICE';
  };

  // Validate coordinates to ensure they're within reasonable bounds
  const isValidCoordinate = (lat: number, lon: number): boolean => {
    return (
      typeof lat === 'number' && 
      typeof lon === 'number' &&
      lat >= -90 && lat <= 90 &&
      lon >= -180 && lon <= 180 &&
      !isNaN(lat) && !isNaN(lon)
    );
  };

  // Focus on a specific vehicle by ID
  const focusOnVehicle = (vehicleId: string) => {
    const vehicle = validVehicles.find(v => v.vehicle_id === vehicleId);
    if (!vehicle || !mapRef.current) {
      return;
    }

    const { lat, lon } = vehicle.location;
    
    let targetLat = lat;
    let targetLon = lon;
    let usingDefault = false;
    
    // Validate coordinates before attempting to focus
    if (!isValidCoordinate(lat, lon)) {
      // Use a default location (London) for invalid coordinates
      targetLat = 51.5074;
      targetLon = -0.1278;
      usingDefault = true;
    }
    
    // Create a temporary popup marker at the target location
    const vehicleType = getVehicleType(vehicle);
    const icon = vehicleType === 'EV' ? evIcon : iceIcon;
    
    // Remove any existing focus marker
    const existingFocusMarker = (mapRef.current as any)._focusMarker;
    if (existingFocusMarker) {
      mapRef.current.removeLayer(existingFocusMarker);
    }
    
    // Create a new focus marker
    const speedStr = format1(vehicle.speed);
    const emissionsStr = format1(vehicle.emissions);
    const batteryStr = vehicleType === 'EV' ? format0(vehicle.battery_level as number) : undefined;
    const fuelStr = vehicleType === 'ICE' ? format0(vehicle.fuel_level as number) : undefined;

    const focusMarker = L.marker([targetLat, targetLon], { icon })
      .bindPopup(`
        <div style="min-width: 200px;">
          <h3 style="margin: 0 0 10px 0; color: ${vehicleType === 'EV' ? '#4CAF50' : '#FF9800'};">
            🚗 Vehicle ${vehicle.vehicle_id}
          </h3>
          <p><strong>Type:</strong> ${vehicleType}</p>
          <p><strong>Speed:</strong> ${speedStr} km/h</p>
          <p><strong>Status:</strong> ${vehicle.status}</p>
          ${vehicleType === 'EV' ? `<p><strong>Battery:</strong> ${batteryStr}%</p>` : `<p><strong>Fuel:</strong> ${fuelStr}%</p>`}
          <p><strong>Emissions:</strong> ${emissionsStr} g/km</p>
          <p><strong>Original Location:</strong> ${vehicle.location.lat.toFixed(4)}, ${vehicle.location.lon.toFixed(4)}</p>
          ${usingDefault ? '<p style="color: orange;"><strong>⚠️ Using default location due to invalid coordinates</strong></p>' : ''}
        </div>
      `, { 
        closeButton: true,
        autoClose: false,
        closeOnClick: false
      });
    
    // Store the focus marker reference
    (mapRef.current as any)._focusMarker = focusMarker;
    
    // Add marker to map
    focusMarker.addTo(mapRef.current);
    
    // Fly to location and open popup
    mapRef.current.flyTo([targetLat, targetLon], 10, {
      animate: true,
      duration: 1.5
    });

    // Open popup after animation
    setTimeout(() => {
      focusMarker.openPopup();
    }, 1800);
  };

  // Expose the focusOnVehicle function via ref
  useImperativeHandle(ref, () => ({
    focusOnVehicle
  }));

  return (
    <Box sx={{ position: 'relative', height: '500px', width: '100%', borderRadius: '8px', overflow: 'hidden' }}>
      {/* Map Style Switcher */}
      <Box sx={{ 
        position: 'absolute', 
        top: 10, 
        right: 10, 
        zIndex: 1000,
        backgroundColor: 'rgba(255, 255, 255, 0.95)',
        borderRadius: 1,
        p: 1,
        boxShadow: 2
      }}>
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel id="map-style-label">Map Style</InputLabel>
          <Select
            labelId="map-style-label"
            value={selectedStyleId}
            label="Map Style"
            onChange={handleStyleChange}
            sx={{ 
              fontSize: '0.875rem',
              '& .MuiOutlinedInput-notchedOutline': {
                borderColor: 'rgba(0, 0, 0, 0.23)',
              },
              '&:hover .MuiOutlinedInput-notchedOutline': {
                borderColor: 'rgba(0, 0, 0, 0.87)',
              },
            }}
          >
            {MAP_STYLES.map((style) => (
              <MenuItem 
                key={style.id} 
                value={style.id}
                sx={{
                  ...(style.id === 'carto-voyager' && {
                    backgroundColor: 'rgba(255, 193, 7, 0.1)',
                    borderLeft: '4px solid #FFC107',
                    '&:hover': {
                      backgroundColor: 'rgba(255, 193, 7, 0.2)',
                    },
                    '&.Mui-selected': {
                      backgroundColor: 'rgba(255, 193, 7, 0.3)',
                      '&:hover': {
                        backgroundColor: 'rgba(255, 193, 7, 0.4)',
                      },
                    },
                  }),
                  ...(style.id === 'carto-minimal' && {
                    backgroundColor: 'rgba(158, 158, 158, 0.1)',
                    borderLeft: '4px solid #9E9E9E',
                    '&:hover': {
                      backgroundColor: 'rgba(158, 158, 158, 0.2)',
                    },
                    '&.Mui-selected': {
                      backgroundColor: 'rgba(158, 158, 158, 0.3)',
                      '&:hover': {
                        backgroundColor: 'rgba(158, 158, 158, 0.4)',
                      },
                    },
                  }),
                }}
              >
                {style.id === 'carto-voyager' ? (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    🎨 {style.name}
                  </Box>
                ) : style.id === 'carto-minimal' ? (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    🗺️ {style.name}
                  </Box>
                ) : (
                  style.name
                )}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      <MapContainer
        center={[20, 0]}
        zoom={2}
        minZoom={2}
        maxZoom={18}
        maxBounds={[[-90, -180], [90, 180]]}
        maxBoundsViscosity={1.0}
        style={{ height: '100%', width: '100%' }}
        scrollWheelZoom={true}
      >
        {/* Capture map instance */}
        <MapInstanceCapture mapRef={mapRef} />
        
        {/* Dynamic Tile Layer */}
        <DynamicTileLayer selectedStyle={selectedStyle} />
        
        {validVehicles.map((vehicle, index) => {
          const vehicleType = getVehicleType(vehicle);
          const icon = vehicleType === 'EV' ? evIcon : iceIcon;
          
          return (
            <VehicleMarker
              key={vehicle.vehicle_id || index}
              vehicle={vehicle}
              icon={icon}
              markersRef={markersRef}
            >
              <Popup>
                <div style={{ minWidth: '200px' }}>
                  <h3 style={{ margin: '0 0 10px 0', color: vehicleType === 'EV' ? '#4CAF50' : '#FF9800' }}>
                    🚗 Vehicle {vehicle.vehicle_id}
                  </h3>
                  <p><strong>Type:</strong> {vehicleType}</p>
                  <p><strong>Speed:</strong> {format1(vehicle.speed)} km/h</p>
                  <p><strong>Status:</strong> {vehicle.status}</p>
                  {vehicleType === 'EV' ? (
                    <>
                      <p><strong>Battery:</strong> {format0(vehicle.battery_level)}%</p>
                      <p><strong>Emissions:</strong> {format1(vehicle.emissions)} g/km</p>
                    </>
                  ) : (
                    <>
                      <p><strong>Fuel:</strong> {format0(vehicle.fuel_level)}%</p>
                      <p><strong>Emissions:</strong> {format1(vehicle.emissions)} g/km</p>
                    </>
                  )}
                  <p><strong>Location:</strong> {vehicle.location.lat.toFixed(4)}, {vehicle.location.lon.toFixed(4)}</p>
                  {onNavigateToVehicleDetail && (
                    <p style={{ marginTop: '10px' }}>
                      <button 
                        onClick={() => onNavigateToVehicleDetail(vehicle.vehicle_id)}
                        style={{
                          background: '#1976d2',
                          color: 'white',
                          border: 'none',
                          padding: '8px 16px',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          fontSize: '14px'
                        }}
                      >
                        View Details
                      </button>
                    </p>
                  )}
                </div>
              </Popup>
            </VehicleMarker>
          );
        })}
      </MapContainer>
    </Box>
  );
});

WorldMap.displayName = 'WorldMap';

export default WorldMap;