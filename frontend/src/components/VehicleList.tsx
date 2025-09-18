import React, { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Typography,
  Box,
  Chip,
  IconButton,
  Collapse,
  TablePagination,
} from '@mui/material';
import {
  KeyboardArrowDown as ExpandMoreIcon,
  KeyboardArrowUp as ExpandLessIcon,
  DirectionsCar as CarIcon,
  BatteryChargingFull as EvIcon,
  LocalGasStation as GasIcon,
  BatteryFull as BatteryIcon,
} from '@mui/icons-material';
import { Telemetry } from '../types';

interface VehicleListProps {
  telemetry: Telemetry[];
  onVehicleFocus?: (vehicleId: string) => void;
}

interface VehicleRowProps {
  vehicle: Telemetry;
  onVehicleFocus?: (vehicleId: string) => void;
}

const VehicleRow: React.FC<VehicleRowProps> = ({ vehicle, onVehicleFocus }) => {
  const [expanded, setExpanded] = useState(false);

  const getVehicleType = (vehicle: Telemetry): 'ICE' | 'EV' => {
    return vehicle.battery_level !== undefined ? 'EV' : 'ICE';
  };

  const getVehicleIcon = (type: 'ICE' | 'EV') => {
    return type === 'EV' ? (
      <EvIcon color="success" fontSize="small" />
    ) : (
      <CarIcon color="primary" fontSize="small" />
    );
  };

  const getStatusColor = (status: 'active' | 'inactive') => {
    return status === 'active' ? 'success' : 'error';
  };

  const getEmissionsColor = (emissions: number) => {
    if (emissions < 10) return 'success';
    if (emissions < 25) return 'warning';
    return 'error';
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const formatLocation = (location: { lat: number; lon: number }) => {
    return `${location.lat.toFixed(4)}, ${location.lon.toFixed(4)}`;
  };

  const vehicleType = getVehicleType(vehicle);

  return (
    <>
      <TableRow 
        hover 
        onClick={() => {
          setExpanded(!expanded);
          if (onVehicleFocus) {
            onVehicleFocus(vehicle.vehicle_id);
          }
        }}
        sx={{ cursor: 'pointer' }}
      >
        <TableCell>
          <IconButton
            size="small"
            onClick={(e) => {
              e.stopPropagation();
              setExpanded(!expanded);
            }}
          >
            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          </IconButton>
        </TableCell>
        <TableCell>
          <Box display="flex" alignItems="center">
            {getVehicleIcon(vehicleType)}
            <Typography variant="body2" ml={1}>
              {vehicle.vehicle_id}
            </Typography>
          </Box>
        </TableCell>
        <TableCell>
          <Chip
            label={vehicleType}
            size="small"
            color={vehicleType === 'EV' ? 'success' : 'primary'}
          />
        </TableCell>
        <TableCell>
          <Chip
            label={vehicle.status}
            size="small"
            color={getStatusColor(vehicle.status)}
          />
        </TableCell>
        <TableCell>
          <Typography variant="body2">
            {(Math.round(vehicle.speed * 10) / 10).toFixed(1)} km/h
          </Typography>
        </TableCell>
        <TableCell>
          <Chip
            label={`${(Math.round(vehicle.emissions * 10) / 10).toFixed(1)} kg`}
            size="small"
            color={getEmissionsColor(vehicle.emissions)}
          />
        </TableCell>
        <TableCell>
          <Typography variant="body2">
            {formatTimestamp(vehicle.timestamp)}
          </Typography>
        </TableCell>
      </TableRow>
      
      <TableRow>
        <TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={7}>
          <Collapse in={expanded} timeout="auto" unmountOnExit>
            <Box sx={{ margin: 1 }}>
              <Typography variant="h6" gutterBottom component="div">
                Vehicle Details
              </Typography>
              <Box display="flex" flexWrap="wrap" gap={2}>
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">
                    Location
                  </Typography>
                  <Typography variant="body2">
                    {formatLocation(vehicle.location)}
                  </Typography>
                </Box>
                
                {vehicle.fuel_level !== undefined && (
                  <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                      <GasIcon fontSize="small" sx={{ mr: 0.5, verticalAlign: 'middle' }} />
                      Fuel Level
                    </Typography>
                    <Typography variant="body2">
                      {Math.round(vehicle.fuel_level)}%
                    </Typography>
                  </Box>
                )}
                
                {vehicle.battery_level !== undefined && (
                  <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                      <BatteryIcon fontSize="small" sx={{ mr: 0.5, verticalAlign: 'middle' }} />
                      Battery Level
                    </Typography>
                    <Typography variant="body2">
                      {Math.round(vehicle.battery_level)}%
                    </Typography>
                  </Box>
                )}
                
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">
                    Emissions Efficiency
                  </Typography>
                  <Typography variant="body2">
                    {Number.isFinite(vehicle.emissions / Math.max(vehicle.speed, 0.1)) ? ((Math.round(((vehicle.emissions / Math.max(vehicle.speed, 0.1)) * 100) * 10) / 10).toFixed(1)) : '0.0'} kg/100km
                  </Typography>
                </Box>
              </Box>
            </Box>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
};

const VehicleList: React.FC<VehicleListProps> = ({ telemetry, onVehicleFocus }) => {
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);

  const handleChangePage = (event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  if (telemetry.length === 0) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" py={4}>
        <Typography variant="body1" color="text.secondary">
          No vehicle data available
        </Typography>
      </Box>
    );
  }

  const paginatedTelemetry = telemetry.slice(
    page * rowsPerPage,
    page * rowsPerPage + rowsPerPage
  );

  return (
    <Box>
      <TableContainer component={Paper} sx={{ overflowX: 'auto' }}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell />
              <TableCell>Vehicle ID</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Speed</TableCell>
              <TableCell>Emissions</TableCell>
              <TableCell>Last Updated</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {paginatedTelemetry.map((vehicle, index) => (
              <VehicleRow key={`${vehicle.vehicle_id}-${index}`} vehicle={vehicle} onVehicleFocus={onVehicleFocus} />
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      
      <TablePagination
        rowsPerPageOptions={[5, 10, 25]}
        component="div"
        count={telemetry.length}
        rowsPerPage={rowsPerPage}
        page={page}
        onPageChange={handleChangePage}
        onRowsPerPageChange={handleChangeRowsPerPage}
      />
    </Box>
  );
};

export default VehicleList; 