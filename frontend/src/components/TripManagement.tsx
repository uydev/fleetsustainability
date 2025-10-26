import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Alert,
  Snackbar,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import TripForm from './TripForm';
import apiService from '../services/api';
import { Trip } from '../types';

interface TripManagementProps {
  timeRange?: { from?: string; to?: string };
}

const TripManagement: React.FC<TripManagementProps> = ({ timeRange }) => {
  const [trips, setTrips] = useState<Trip[]>([]);
  const [loading, setLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error';
  }>({ open: false, message: '', severity: 'success' });

  const loadTrips = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiService.getTrips(timeRange);
      setTrips(data);
    } catch (error) {
      console.error('Error loading trips:', error);
      setSnackbar({
        open: true,
        message: 'Failed to load trip data',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    loadTrips();
  }, [loadTrips]);

  const handleAddTrip = async (tripData: Omit<Trip, 'id'>) => {
    try {
      await apiService.postTrip(tripData);
      setSnackbar({
        open: true,
        message: 'Trip created successfully',
        severity: 'success',
      });
      loadTrips();
    } catch (error) {
      console.error('Error creating trip:', error);
      setSnackbar({
        open: true,
        message: 'Failed to create trip',
        severity: 'error',
      });
    }
  };

  const handleDeleteTrip = async (id: string) => {
    const ok = window.confirm('Delete this trip?');
    if (!ok) return;
    try {
      await apiService.deleteTrip(id);
      setSnackbar({ open: true, message: 'Trip deleted', severity: 'success' });
      loadTrips();
    } catch (error) {
      console.error('Error deleting trip:', error);
      setSnackbar({ open: true, message: 'Failed to delete trip', severity: 'error' });
    }
  };

  const openAddForm = () => {
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'in_progress': return 'warning';
      case 'planned': return 'info';
      case 'cancelled': return 'error';
      default: return 'default';
    }
  };

  const getPurposeColor = (purpose: string) => {
    switch (purpose) {
      case 'business': return 'primary';
      case 'delivery': return 'secondary';
      case 'personal': return 'default';
      default: return 'default';
    }
  };

  const formatDuration = (hours: number) => {
    const h = Math.floor(hours);
    const m = Math.round((hours - h) * 60);
    return `${h}h ${m}m`;
  };

  const formatDateTime = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5" component="h2">
          Trip Management
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={openAddForm}
        >
          Add Trip
        </Button>
      </Box>

      <Paper sx={{ p: 2 }}>
        {loading ? (
          <Typography>Loading trip data...</Typography>
        ) : trips.length === 0 ? (
          <Typography color="text.secondary" align="center" py={4}>
            No trips found. Add your first trip to get started.
          </Typography>
        ) : (
          <TableContainer sx={{ overflowX: 'auto' }}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Vehicle ID</TableCell>
                  <TableCell>Purpose</TableCell>
                  <TableCell>Start Time</TableCell>
                  <TableCell>End Time</TableCell>
                  <TableCell>Distance (km)</TableCell>
                  <TableCell>Duration</TableCell>
                  <TableCell>Cost ($)</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {trips.map((trip) => (
                  <TableRow key={trip.id}>
                    <TableCell>{trip.vehicle_id}</TableCell>
                    <TableCell>
                      <Chip
                        label={trip.purpose}
                        color={getPurposeColor(trip.purpose) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>{formatDateTime(trip.start_time)}</TableCell>
                    <TableCell>{formatDateTime(trip.end_time)}</TableCell>
                    <TableCell>{trip.distance.toFixed(1)}</TableCell>
                    <TableCell>{formatDuration(trip.duration)}</TableCell>
                    <TableCell>${trip.cost.toFixed(2)}</TableCell>
                    <TableCell>
                      <Chip
                        label={trip.status.replace('_', ' ')}
                        color={getStatusColor(trip.status) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Tooltip title="Delete">
                        <IconButton size="small" onClick={() => handleDeleteTrip(trip.id!)}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      <TripForm
        open={formOpen}
        onClose={closeForm}
        onSubmit={handleAddTrip}
        title="Add Trip"
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

export default TripManagement; 