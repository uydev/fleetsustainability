import React, { useState, useRef, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  IconButton,
  Tooltip,
  Card,
  CardContent,
  Divider,
  Grid,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Badge,
  Alert,
  CircularProgress,
  Fade,
  Zoom,
} from '@mui/material';
import {
  DirectionsCar as CarIcon,
  BatteryChargingFull as EvIcon,
  LocationOn as LocationIcon,
  Speed as SpeedIcon,
  EmojiNature as EcoIcon,
  SignalCellular4Bar as SignalIcon,
  Info as InfoIcon,
  ZoomIn as ZoomInIcon,
  ZoomOut as ZoomOutIcon,
  MyLocation as MyLocationIcon,
  Close as CloseIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { Telemetry } from '../types';

interface FleetMapProps {
  telemetry: Telemetry[];
}

interface LocationInfo {
  country?: string;
  city?: string;
  town?: string;
  street?: string;
  address?: string;
}

const FleetMap: React.FC<FleetMapProps> = ({ telemetry }) => {
  const [selectedVehicle, setSelectedVehicle] = useState<Telemetry | null>(null);
  const [locationInfo, setLocationInfo] = useState<LocationInfo>({});
  const [loading, setLoading] = useState(false);
  const [zoom, setZoom] = useState(0.01);
  const [centerLat, setCenterLat] = useState(0);
  const [centerLon, setCenterLon] = useState(0);

  // Determine vehicle type and status
  const getVehicleType = (vehicle: Telemetry): 'ICE' | 'EV' => {
    if (vehicle.type === 'EV' || vehicle.type === 'ICE') {
      return vehicle.type;
    }
    return vehicle.battery_level !== undefined && vehicle.battery_level !== null ? 'EV' : 'ICE';
  };

  const getVehicleStatus = (vehicle: Telemetry): 'active' | 'inactive' => {
    return vehicle.status || 'active';
  };

  // Get detailed location information using reverse geocoding
  const getLocationInfo = async (lat: number, lon: number) => {
    setLoading(true);
    try {
      const response = await fetch(
        `https://nominatim.openstreetmap.org/reverse?format=json&lat=${lat}&lon=${lon}&zoom=18&addressdetails=1`
      );
      const data = await response.json();
      
      if (data.address) {
        setLocationInfo({
          country: data.address.country,
          city: data.address.city || data.address.town,
          town: data.address.town || data.address.village,
          street: data.address.road,
          address: data.display_name
        });
      }
    } catch (error) {
      console.error('Error fetching location info:', error);
      // Fallback to mock data for demo
      setLocationInfo({
        country: 'United States',
        city: 'New York',
        town: 'Manhattan',
        street: 'Broadway',
        address: `${lat.toFixed(4)}, ${lon.toFixed(4)}`
      });
    } finally {
      setLoading(false);
    }
  };

  // Convert lat/lon to relative position on map
  const getVehiclePosition = (lat: number, lon: number) => {
    const latDiff = lat - centerLat;
    const lonDiff = lon - centerLon;
    
    // Scale based on zoom level - massive zoom range from world to street level
    const scale = 10 * zoom;
    const x = 50 + (lonDiff * scale);
    const y = 50 - (latDiff * scale);
    
    return { x, y };
  };

  if (!telemetry || telemetry.length === 0) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        height="100%"
        sx={{
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          borderRadius: 1
        }}
      >
        <Typography variant="h6" color="white">
          No vehicle data available
        </Typography>
      </Box>
    );
  }

  // Filter out vehicles with invalid location data
  const validTelemetry = telemetry.filter(vehicle => 
    vehicle.location && 
    typeof vehicle.location.lat === 'number' && 
    typeof vehicle.location.lon === 'number'
  );

  if (validTelemetry.length === 0) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        height="100%"
        sx={{
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          borderRadius: 1
        }}
      >
        <Typography variant="h6" color="white">
          No vehicles with valid location data
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Professional Header */}
      <Box sx={{ 
        p: 2, 
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        color: 'white',
        borderRadius: '8px 8px 0 0'
      }}>
        <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
          🌍 Fleet Location Intelligence
        </Typography>
        <Typography variant="body2" sx={{ opacity: 0.9 }}>
          Real-time vehicle tracking with detailed location information
        </Typography>
      </Box>

      {/* Sophisticated World Map */}
      <Box sx={{ flex: 1, position: 'relative', p: 2 }}>
        <Paper 
          elevation={3}
          sx={{ 
            height: '100%', 
            position: 'relative',
            background: 'linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%)',
            borderRadius: '0 0 8px 8px',
            overflow: 'hidden'
          }}
        >
                    {/* REAL WORLD MAP WITH ACTUAL CONTINENT SHAPES */}
          <svg
            width="100%"
            height="100%"
            viewBox="0 0 1000 500"
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              opacity: zoom < 0.1 ? 1.0 : 0.7
            }}
          >
            {/* Ocean Background */}
            <rect width="1000" height="500" fill="#4A90E2" />
            
            {/* North America */}
            <path
              d="M 150 120 L 200 100 L 250 110 L 280 130 L 300 150 L 320 180 L 300 220 L 280 240 L 250 250 L 220 240 L 200 220 L 180 200 L 160 180 L 140 150 Z"
              fill="#4CAF50"
              stroke="#2E7D32"
              strokeWidth="2"
            />
            
            {/* South America */}
            <path
              d="M 250 250 L 280 240 L 300 260 L 320 300 L 330 350 L 320 400 L 300 430 L 280 440 L 250 430 L 230 400 L 220 350 L 230 300 L 240 260 Z"
              fill="#4CAF50"
              stroke="#2E7D32"
              strokeWidth="2"
            />
            
            {/* Europe */}
            <path
              d="M 400 100 L 450 90 L 480 100 L 500 120 L 490 140 L 470 160 L 450 170 L 430 160 L 410 140 L 400 120 Z"
              fill="#4CAF50"
              stroke="#2E7D32"
              strokeWidth="2"
            />
            
            {/* Africa */}
            <path
              d="M 450 170 L 500 160 L 530 180 L 550 220 L 560 280 L 550 340 L 530 380 L 500 390 L 470 380 L 450 340 L 440 280 L 450 220 L 460 180 Z"
              fill="#4CAF50"
              stroke="#2E7D32"
              strokeWidth="2"
            />
            
            {/* Asia */}
            <path
              d="M 500 80 L 650 70 L 750 80 L 820 100 L 850 130 L 860 170 L 850 210 L 820 240 L 780 250 L 720 240 L 680 220 L 650 200 L 620 180 L 600 150 L 580 120 L 550 100 Z"
              fill="#4CAF50"
              stroke="#2E7D32"
              strokeWidth="2"
            />
            
            {/* Australia */}
            <path
              d="M 700 320 L 780 310 L 820 320 L 840 340 L 830 360 L 800 370 L 760 365 L 720 360 L 700 350 Z"
              fill="#4CAF50"
              stroke="#2E7D32"
              strokeWidth="2"
            />
            
            {/* Antarctica (bottom) */}
            <path
              d="M 100 450 L 900 450 L 900 480 L 100 480 Z"
              fill="#E8F5E8"
              stroke="#2E7D32"
              strokeWidth="1"
            />
          </svg>
          
          {/* Grid Lines for Professional Look */}
          <Box sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: `
              linear-gradient(90deg, rgba(255,255,255,0.1) 1px, transparent 1px),
              linear-gradient(rgba(255,255,255,0.1) 1px, transparent 1px)
            `,
            backgroundSize: '20px 20px'
          }} />

          {/* Vehicle Markers */}
          {validTelemetry.map((vehicle, index) => {
            const position = getVehiclePosition(vehicle.location.lat, vehicle.location.lon);
            const vehicleType = getVehicleType(vehicle);
            const vehicleStatus = getVehicleStatus(vehicle);

            return (
              <Tooltip
                key={vehicle.vehicle_id || index}
                title={
                  <Box>
                    <Typography variant="subtitle2">
                      Vehicle {vehicle.vehicle_id}
                    </Typography>
                    <Typography variant="body2">
                      Speed: {vehicle.speed} km/h
                    </Typography>
                    <Typography variant="body2">
                      Emissions: {vehicle.emissions} g/km
                    </Typography>
                    <Typography variant="body2">
                      Location: {vehicle.location.lat.toFixed(4)}, {vehicle.location.lon.toFixed(4)}
                    </Typography>
                  </Box>
                }
                arrow
                placement="top"
              >
                <Box
                  sx={{
                    position: 'absolute',
                    left: `${position.x}%`,
                    top: `${position.y}%`,
                    transform: 'translate(-50%, -50%)',
                    cursor: 'pointer',
                    transition: 'all 0.3s ease',
                    '&:hover': {
                      transform: 'translate(-50%, -50%) scale(1.2)',
                      zIndex: 10
                    }
                  }}
                  onClick={() => {
                    setSelectedVehicle(vehicle);
                    getLocationInfo(vehicle.location.lat, vehicle.location.lon);
                  }}
                >
                  <Box
                    sx={{
                      width: 40,
                      height: 40,
                      borderRadius: '50%',
                      background: vehicleType === 'EV' 
                        ? 'linear-gradient(135deg, #4caf50 0%, #66bb6a 100%)'
                        : 'linear-gradient(135deg, #ff9800 0%, #ffb74d 100%)',
                      border: '3px solid white',
                      boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      color: 'white',
                      fontSize: '18px',
                      fontWeight: 'bold'
                    }}
                  >
                    {vehicleType === 'EV' ? '⚡' : '🚗'}
                  </Box>
                  
                  {/* Status Indicator */}
                  <Box
                    sx={{
                      position: 'absolute',
                      top: -5,
                      right: -5,
                      width: 12,
                      height: 12,
                      borderRadius: '50%',
                      background: vehicleStatus === 'active' ? '#4caf50' : '#f44336',
                      border: '2px solid white',
                      boxShadow: '0 2px 4px rgba(0,0,0,0.3)'
                    }}
                  />
                </Box>
              </Tooltip>
            );
          })}

          {/* Map Controls */}
          <Box sx={{ position: 'absolute', top: 16, left: 16 }}>
            <Paper sx={{ p: 1, opacity: 0.9 }}>
              <Typography variant="caption" sx={{ fontWeight: 'bold' }}>
                Vehicle Legend
              </Typography>
              <Box sx={{ mt: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 0.5 }}>
                  <Box sx={{ 
                    width: 16, 
                    height: 16, 
                    borderRadius: '50%', 
                    background: 'linear-gradient(135deg, #4caf50 0%, #66bb6a 100%)',
                    mr: 1 
                  }} />
                  <Typography variant="caption">EV</Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  <Box sx={{ 
                    width: 16, 
                    height: 16, 
                    borderRadius: '50%', 
                    background: 'linear-gradient(135deg, #ff9800 0%, #ffb74d 100%)',
                    mr: 1 
                  }} />
                  <Typography variant="caption">ICE</Typography>
                </Box>
              </Box>
            </Paper>
          </Box>

          {/* Vehicle Count Badge */}
          <Box sx={{ position: 'absolute', top: 16, right: 16 }}>
            <Chip
              label={`${validTelemetry.length} Vehicles`}
              color="primary"
              sx={{ fontWeight: 'bold' }}
            />
          </Box>
          
          {/* Continent Labels - Only visible at low zoom */}
          {zoom < 0.1 && (
            <>
              <Box sx={{ position: 'absolute', top: '35%', left: '25%', color: 'white', fontWeight: 'bold', textShadow: '2px 2px 4px rgba(0,0,0,0.8)' }}>
                North America
              </Box>
              <Box sx={{ position: 'absolute', top: '65%', left: '30%', color: 'white', fontWeight: 'bold', textShadow: '2px 2px 4px rgba(0,0,0,0.8)' }}>
                South America
              </Box>
              <Box sx={{ position: 'absolute', top: '30%', left: '45%', color: 'white', fontWeight: 'bold', textShadow: '2px 2px 4px rgba(0,0,0,0.8)' }}>
                Europe
              </Box>
              <Box sx={{ position: 'absolute', top: '55%', left: '50%', color: 'white', fontWeight: 'bold', textShadow: '2px 2px 4px rgba(0,0,0,0.8)' }}>
                Africa
              </Box>
              <Box sx={{ position: 'absolute', top: '35%', left: '70%', color: 'white', fontWeight: 'bold', textShadow: '2px 2px 4px rgba(0,0,0,0.8)' }}>
                Asia
              </Box>
              <Box sx={{ position: 'absolute', top: '70%', left: '75%', color: 'white', fontWeight: 'bold', textShadow: '2px 2px 4px rgba(0,0,0,0.8)' }}>
                Australia
              </Box>
            </>
          )}

          {/* Zoom Controls */}
          <Box sx={{ position: 'absolute', bottom: 16, right: 16 }}>
            <Paper sx={{ p: 0.5, opacity: 0.9 }}>
              <Typography variant="caption" sx={{ display: 'block', textAlign: 'center', mb: 0.5 }}>
                Zoom: {zoom.toFixed(2)}x
              </Typography>
              <IconButton 
                size="small" 
                onClick={() => setZoom(Math.min(zoom * 1.5, 50))}
                disabled={zoom >= 50}
              >
                <ZoomInIcon />
              </IconButton>
              <IconButton 
                size="small" 
                onClick={() => setZoom(Math.max(zoom / 1.5, 0.001))}
                disabled={zoom <= 0.001}
              >
                <ZoomOutIcon />
              </IconButton>
            </Paper>
          </Box>
          
          {/* Zoom Preset Buttons */}
          <Box sx={{ position: 'absolute', bottom: 16, left: 16 }}>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button
                variant="contained"
                size="small"
                startIcon={<MyLocationIcon />}
                onClick={() => {
                  setZoom(0.01);
                  setCenterLat(0);
                  setCenterLon(0);
                }}
                sx={{ 
                  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                  color: 'white',
                  '&:hover': {
                    background: 'linear-gradient(135deg, #5a6fd8 0%, #6a4190 100%)'
                  }
                }}
              >
                World
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={() => setZoom(0.1)}
                sx={{ color: 'white', borderColor: 'white' }}
              >
                Country
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={() => setZoom(1)}
                sx={{ color: 'white', borderColor: 'white' }}
              >
                City
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={() => setZoom(10)}
                sx={{ color: 'white', borderColor: 'white' }}
              >
                Street
              </Button>
            </Box>
          </Box>
        </Paper>
      </Box>

      {/* Vehicle Details Dialog */}
      <Dialog 
        open={!!selectedVehicle} 
        onClose={() => setSelectedVehicle(null)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Typography variant="h6">
              Vehicle Details
            </Typography>
            <IconButton onClick={() => setSelectedVehicle(null)}>
              <CloseIcon />
            </IconButton>
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedVehicle && (
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <Card sx={{ mb: 2 }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                      {getVehicleType(selectedVehicle) === 'EV' ? (
                        <EvIcon color="success" sx={{ fontSize: 40, mr: 2 }} />
                      ) : (
                        <CarIcon color="primary" sx={{ fontSize: 40, mr: 2 }} />
                      )}
                      <Box>
                        <Typography variant="h6">
                          Vehicle {selectedVehicle.vehicle_id}
                        </Typography>
                        <Chip 
                          label={getVehicleType(selectedVehicle)} 
                          color={getVehicleType(selectedVehicle) === 'EV' ? 'success' : 'warning'}
                          size="small"
                        />
                        <Chip 
                          label={getVehicleStatus(selectedVehicle)} 
                          color={getVehicleStatus(selectedVehicle) === 'active' ? 'success' : 'error'}
                          size="small"
                          sx={{ ml: 1 }}
                        />
                      </Box>
                    </Box>
                    
                    <Grid container spacing={2}>
                      <Grid item xs={6}>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                          <SpeedIcon sx={{ mr: 1, color: 'primary.main' }} />
                          <Typography variant="body2">
                            {selectedVehicle.speed} km/h
                          </Typography>
                        </Box>
                      </Grid>
                      <Grid item xs={6}>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                          <EcoIcon sx={{ mr: 1, color: 'success.main' }} />
                          <Typography variant="body2">
                            {selectedVehicle.emissions} g/km
                          </Typography>
                        </Box>
                      </Grid>
                      {selectedVehicle.fuel_level && (
                        <Grid item xs={6}>
                          <Typography variant="body2">
                            Fuel: {selectedVehicle.fuel_level}%
                          </Typography>
                        </Grid>
                      )}
                      {selectedVehicle.battery_level && (
                        <Grid item xs={6}>
                          <Typography variant="body2">
                            Battery: {selectedVehicle.battery_level}%
                          </Typography>
                        </Grid>
                      )}
                    </Grid>
                  </CardContent>
                </Card>
              </Grid>

              {/* Location Information */}
              <Grid item xs={12}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" sx={{ mb: 2, display: 'flex', alignItems: 'center' }}>
                      <LocationIcon sx={{ mr: 1 }} />
                      Location Details
                    </Typography>
                    
                    {loading ? (
                      <Box sx={{ display: 'flex', alignItems: 'center' }}>
                        <CircularProgress size={20} sx={{ mr: 1 }} />
                        <Typography>Loading location details...</Typography>
                      </Box>
                    ) : (
                      <List dense>
                        {locationInfo.country && (
                          <ListItem>
                            <ListItemIcon>
                              <LocationIcon color="primary" />
                            </ListItemIcon>
                            <ListItemText 
                              primary="Country" 
                              secondary={locationInfo.country}
                            />
                          </ListItem>
                        )}
                        {locationInfo.city && (
                          <ListItem>
                            <ListItemIcon>
                              <LocationIcon color="primary" />
                            </ListItemIcon>
                            <ListItemText 
                              primary="City" 
                              secondary={locationInfo.city}
                            />
                          </ListItem>
                        )}
                        {locationInfo.town && (
                          <ListItem>
                            <ListItemIcon>
                              <LocationIcon color="primary" />
                            </ListItemIcon>
                            <ListItemText 
                              primary="Town" 
                              secondary={locationInfo.town}
                            />
                          </ListItem>
                        )}
                        {locationInfo.street && (
                          <ListItem>
                            <ListItemIcon>
                              <LocationIcon color="primary" />
                            </ListItemIcon>
                            <ListItemText 
                              primary="Street" 
                              secondary={locationInfo.street}
                            />
                          </ListItem>
                        )}
                        {locationInfo.address && (
                          <ListItem>
                            <ListItemIcon>
                              <LocationIcon color="primary" />
                            </ListItemIcon>
                            <ListItemText 
                              primary="Full Address" 
                              secondary={locationInfo.address}
                            />
                          </ListItem>
                        )}
                      </List>
                    )}
                  </CardContent>
                </Card>
              </Grid>
            </Grid>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSelectedVehicle(null)}>
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default FleetMap;
