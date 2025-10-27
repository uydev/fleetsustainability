import React, { useState, useEffect, useCallback } from 'react';
import { Box, Button, Paper, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Chip, Alert, Snackbar } from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import TelemetryForm from './TelemetryForm';
import apiService from '../services/api';
import { Telemetry } from '../types';

interface TelemetryManagementProps {
  timeRange?: { from?: string; to?: string };
}

const TelemetryManagement: React.FC<TelemetryManagementProps> = ({ timeRange }) => {
  const [telemetry, setTelemetry] = useState<Telemetry[]>([]);
  const [loading, setLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error';
  }>({ open: false, message: '', severity: 'success' });

  const loadTelemetry = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiService.getTelemetry(timeRange);
      setTelemetry(data);
    } catch (error) {
      console.error('Error loading telemetry:', error);
      setSnackbar({
        open: true,
        message: 'Failed to load telemetry data',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    loadTelemetry();
  }, [loadTelemetry]);

  const handleAddTelemetry = async (telemetryData: Omit<Telemetry, 'id'>) => {
    try {
      await apiService.postTelemetry(telemetryData);
      setSnackbar({
        open: true,
        message: 'Telemetry entry created successfully',
        severity: 'success',
      });
      loadTelemetry();
    } catch (error) {
      console.error('Error creating telemetry:', error);
      setSnackbar({
        open: true,
        message: 'Failed to create telemetry entry',
        severity: 'error',
      });
    }
  };

  const openAddForm = () => {
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
  };

  const getVehicleTypeColor = (telemetry: Telemetry) => {
    return telemetry.battery_level !== undefined ? 'success' : 'warning';
  };

  const getVehicleType = (telemetry: Telemetry) => {
    return telemetry.battery_level !== undefined ? 'EV' : 'ICE';
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5" component="h2">
          Telemetry Management
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={openAddForm}
        >
          Add Telemetry Entry
        </Button>
      </Box>

      <Paper sx={{ p: 2 }}>
        {loading ? (
          <Typography>Loading telemetry data...</Typography>
        ) : telemetry.length === 0 ? (
          <Typography color="text.secondary" align="center" py={4}>
            No telemetry data found. Add your first telemetry entry to get started.
          </Typography>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Vehicle ID</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Timestamp</TableCell>
                  <TableCell>Location</TableCell>
                  <TableCell>Speed (km/h)</TableCell>
                  <TableCell>Fuel/Battery (%)</TableCell>
                  <TableCell>Emissions (kg)</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {telemetry.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>{entry.vehicle_id}</TableCell>
                    <TableCell>
                      <Chip
                        label={getVehicleType(entry)}
                        color={getVehicleTypeColor(entry) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>{formatTimestamp(entry.timestamp)}</TableCell>
                    <TableCell>
                      {entry.location ? (
                        `${entry.location.lat.toFixed(4)}, ${entry.location.lon.toFixed(4)}`
                      ) : (
                        'N/A'
                      )}
                    </TableCell>
                    <TableCell>{(Math.round(entry.speed * 10) / 10).toFixed(1)}</TableCell>
                    <TableCell>
                      {entry.fuel_level !== undefined
                        ? Math.round(entry.fuel_level)
                        : entry.battery_level !== undefined
                        ? Math.round(entry.battery_level)
                        : 'N/A'}
                    </TableCell>
                    <TableCell>{(Math.round(entry.emissions * 10) / 10).toFixed(1)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      <TelemetryForm
        open={formOpen}
        onClose={closeForm}
        onSubmit={handleAddTelemetry}
        title="Add Telemetry Entry"
      />

      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default TelemetryManagement; 