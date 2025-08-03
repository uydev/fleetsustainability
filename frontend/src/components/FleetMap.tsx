import React from 'react';
import {
  Box,
  Typography,
} from '@mui/material';
import {
  DirectionsCar as CarIcon,
  BatteryChargingFull as EvIcon,
} from '@mui/icons-material';
import { Telemetry } from '../types';

interface FleetMapProps {
  telemetry: Telemetry[];
}

const FleetMap: React.FC<FleetMapProps> = ({ telemetry }) => {
  // Simple map visualization - in a real app, you'd use a proper mapping library
  // like Mapbox, Google Maps, or Leaflet
  const centerLat = 40.7128; // New York
  const centerLon = -74.0060;
  const mapSize = 400;

  const getVehicleIcon = (type: 'ICE' | 'EV') => {
    return type === 'EV' ? (
      <EvIcon color="success" fontSize="small" />
    ) : (
      <CarIcon color="primary" fontSize="small" />
    );
  };

  const getVehicleColor = (type: 'ICE' | 'EV') => {
    return type === 'EV' ? '#4caf50' : '#ff9800';
  };

  const getVehicleStatusColor = (status: 'active' | 'inactive') => {
    return status === 'active' ? '#4caf50' : '#f44336';
  };

  // Determine vehicle type based on available data
  const getVehicleType = (vehicle: Telemetry): 'ICE' | 'EV' => {
    // First check if type is explicitly set
    if (vehicle.type === 'EV' || vehicle.type === 'ICE') {
      return vehicle.type;
    }
    // Fallback to battery level detection
    return vehicle.battery_level !== undefined && vehicle.battery_level !== null ? 'EV' : 'ICE';
  };

  // Get vehicle status with fallback
  const getVehicleStatus = (vehicle: Telemetry): 'active' | 'inactive' => {
    return vehicle.status || 'active';
  };

  if (!telemetry || telemetry.length === 0) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        height="100%"
        bgcolor="#f5f5f5"
        borderRadius={1}
      >
        <Typography variant="body1" color="text.secondary">
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

  // Debug logging
  console.log('FleetMap Debug:', {
    totalTelemetry: telemetry.length,
    validTelemetry: validTelemetry.length,
    invalidCount: telemetry.length - validTelemetry.length,
    sampleVehicle: validTelemetry[0] || null
  });

  if (validTelemetry.length === 0) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        height="100%"
        bgcolor="#f5f5f5"
        borderRadius={1}
      >
        <Typography variant="body1" color="text.secondary">
          No vehicles with valid location data
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Simple Map Visualization */}
      <Box
        position="relative"
        width="100%"
        flex={1}
        minHeight={300}
        bgcolor="#e3f2fd"
        borderRadius={1}
        border="2px solid #2196f3"
        overflow="hidden"
      >
        {/* Map Grid */}
        <Box
          position="absolute"
          top={0}
          left={0}
          right={0}
          bottom={0}
          sx={{
            backgroundImage: `
              linear-gradient(rgba(33, 150, 243, 0.1) 1px, transparent 1px),
              linear-gradient(90deg, rgba(33, 150, 243, 0.1) 1px, transparent 1px)
            `,
            backgroundSize: '20px 20px',
          }}
        />

        {/* Vehicle Markers */}
        {validTelemetry.map((vehicle, index) => {
          // Validate required data
          if (!vehicle.location || typeof vehicle.location.lat !== 'number' || typeof vehicle.location.lon !== 'number') {
            console.warn(`Vehicle ${vehicle.vehicle_id} has invalid location data:`, vehicle.location);
            return null;
          }

          // Convert lat/lon to relative position on map
          const latDiff = vehicle.location.lat - centerLat;
          const lonDiff = vehicle.location.lon - centerLon;
          
          // Scale to fit map with better bounds checking
          const x = 50 + (lonDiff * 100); // Reduced scale factor for better visibility
          const y = 50 - (latDiff * 100); // Reduced scale factor for better visibility
          
          const vehicleType = getVehicleType(vehicle);
          const vehicleStatus = getVehicleStatus(vehicle);
          
          return (
            <Box
              key={`${vehicle.vehicle_id}-${index}`}
              sx={{
                position: 'absolute',
                left: `${Math.max(10, Math.min(90, x))}%`, // Increased bounds for better visibility
                top: `${Math.max(10, Math.min(90, y))}%`, // Increased bounds for better visibility
                transform: 'translate(-50%, -50%)',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                cursor: 'pointer',
                '&:hover': {
                  zIndex: 10,
                },
              }}
            >
              <Box
                sx={{
                  color: getVehicleColor(vehicleType),
                  filter: vehicleStatus === 'inactive' ? 'grayscale(100%)' : 'none',
                }}
              >
                {getVehicleIcon(vehicleType)}
              </Box>
              
              {/* Status indicator */}
              <Box
                width={8}
                height={8}
                borderRadius="50%"
                bgcolor={getVehicleStatusColor(vehicleStatus)}
                border="1px solid white"
                position="absolute"
                top={-4}
                right={-4}
              />
              
              {/* Tooltip */}
              <Box
                sx={{
                  position: 'absolute',
                  top: '100%',
                  left: '50%',
                  transform: 'translateX(-50%)',
                  bgcolor: 'rgba(0, 0, 0, 0.8)',
                  color: 'white',
                  px: 1,
                  py: 0.5,
                  borderRadius: 1,
                  fontSize: '0.75rem',
                  whiteSpace: 'nowrap',
                  zIndex: 5,
                  opacity: 0,
                  transition: 'opacity 0.2s',
                  '&:hover': {
                    opacity: 1,
                  },
                }}
              >
                {vehicle.vehicle_id}
                <br />
                Speed: {vehicle.speed?.toFixed(0) || 'N/A'} km/h
                <br />
                Emissions: {vehicle.emissions?.toFixed(1) || 'N/A'} kg
                <br />
                Type: {vehicleType}
                <br />
                Status: {vehicleStatus}
              </Box>
            </Box>
          );
        })}

        {/* Map Legend */}
        <Box
          position="absolute"
          bottom={8}
          right={8}
          bgcolor="rgba(255, 255, 255, 0.9)"
          p={1}
          borderRadius={1}
          fontSize="0.75rem"
        >
          <Box display="flex" alignItems="center" mb={0.5}>
            <EvIcon color="success" fontSize="small" sx={{ mr: 0.5 }} />
            <Typography variant="caption">EV</Typography>
          </Box>
          <Box display="flex" alignItems="center" mb={0.5}>
            <CarIcon color="primary" fontSize="small" sx={{ mr: 0.5 }} />
            <Typography variant="caption">ICE</Typography>
          </Box>
          <Box display="flex" alignItems="center">
            <Box
              width={8}
              height={8}
              borderRadius="50%"
              bgcolor="#4caf50"
              mr={0.5}
            />
            <Typography variant="caption">Active</Typography>
          </Box>
        </Box>
      </Box>

      {/* Fleet Summary */}
      <Box mt={1}>
        <Typography variant="body2" color="text.secondary">
          Fleet Overview: {validTelemetry.length} vehicles
          ({validTelemetry.filter(v => getVehicleType(v) === 'EV').length} EV, 
          {validTelemetry.filter(v => getVehicleType(v) === 'ICE').length} ICE)
        </Typography>
      </Box>
    </Box>
  );
};

export default FleetMap; 