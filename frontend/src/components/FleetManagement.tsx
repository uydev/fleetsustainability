import React, { useState, useEffect } from 'react';
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
  IconButton,
  Chip,
  Alert,
  Snackbar,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import VehicleForm from './VehicleForm';
import apiService from '../services/api';
import { Vehicle } from '../types';

interface FleetManagementProps {
  timeRange?: { from?: string; to?: string };
}

const FleetManagement: React.FC<FleetManagementProps> = ({ timeRange }) => {
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [editingVehicle, setEditingVehicle] = useState<Vehicle | undefined>();
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error';
  }>({ open: false, message: '', severity: 'success' });

  const loadVehicles = async () => {
    setLoading(true);
    try {
      const data = await apiService.getVehicles(timeRange);
      setVehicles(data);
    } catch (error) {
      console.error('Error loading vehicles:', error);
      setSnackbar({
        open: true,
        message: 'Failed to load vehicles',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadVehicles();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [timeRange?.from, timeRange?.to]);

  const handleAddVehicle = async (vehicle: Omit<Vehicle, 'id'>) => {
    try {
      await apiService.createVehicle(vehicle);
      setSnackbar({
        open: true,
        message: 'Vehicle created successfully',
        severity: 'success',
      });
      loadVehicles();
    } catch (error) {
      console.error('Error creating vehicle:', error);
      setSnackbar({
        open: true,
        message: 'Failed to create vehicle',
        severity: 'error',
      });
    }
  };

  const handleEditVehicle = async (vehicle: Omit<Vehicle, 'id'>) => {
    if (!editingVehicle) return;
    
    try {
      await apiService.updateVehicle(editingVehicle.id, vehicle);
      setSnackbar({
        open: true,
        message: 'Vehicle updated successfully',
        severity: 'success',
      });
      loadVehicles();
    } catch (error) {
      console.error('Error updating vehicle:', error);
      setSnackbar({
        open: true,
        message: 'Failed to update vehicle',
        severity: 'error',
      });
    }
  };

  const handleDeleteVehicle = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this vehicle?')) {
      return;
    }

    try {
      await apiService.deleteVehicle(id);
      setSnackbar({
        open: true,
        message: 'Vehicle deleted successfully',
        severity: 'success',
      });
      loadVehicles();
    } catch (error) {
      console.error('Error deleting vehicle:', error);
      setSnackbar({
        open: true,
        message: 'Failed to delete vehicle',
        severity: 'error',
      });
    }
  };

  const openAddForm = () => {
    setEditingVehicle(undefined);
    setFormOpen(true);
  };

  const openEditForm = (vehicle: Vehicle) => {
    setEditingVehicle(vehicle);
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
    setEditingVehicle(undefined);
  };

  const handleFormSubmit = async (vehicle: Omit<Vehicle, 'id'>) => {
    if (editingVehicle) {
      await handleEditVehicle(vehicle);
    } else {
      await handleAddVehicle(vehicle);
    }
  };

  const getVehicleTypeColor = (type: string) => {
    return type === 'EV' ? 'success' : 'warning';
  };

  const getStatusColor = (status: string) => {
    return status === 'active' ? 'success' : 'default';
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5" component="h2">
          Fleet Management
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={openAddForm}
        >
          Add Vehicle
        </Button>
      </Box>

      <Paper sx={{ p: 2 }}>
        {loading ? (
          <Typography>Loading vehicles...</Typography>
        ) : vehicles.length === 0 ? (
          <Typography color="text.secondary" align="center" py={4}>
            No vehicles found. Add your first vehicle to get started.
          </Typography>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Type</TableCell>
                  <TableCell>Make</TableCell>
                  <TableCell>Model</TableCell>
                  <TableCell>Year</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Location</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {vehicles.map((vehicle) => (
                  <TableRow key={vehicle.id}>
                    <TableCell>
                      <Chip
                        label={vehicle.type}
                        color={getVehicleTypeColor(vehicle.type) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>{vehicle.make}</TableCell>
                    <TableCell>{vehicle.model}</TableCell>
                    <TableCell>{vehicle.year}</TableCell>
                    <TableCell>
                      <Chip
                        label={vehicle.status}
                        color={getStatusColor(vehicle.status) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      {vehicle.current_location ? (
                        `${vehicle.current_location.lat.toFixed(4)}, ${vehicle.current_location.lon.toFixed(4)}`
                      ) : (
                        'N/A'
                      )}
                    </TableCell>
                    <TableCell align="right">
                      <IconButton
                        size="small"
                        onClick={() => openEditForm(vehicle)}
                        color="primary"
                      >
                        <EditIcon />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={() => handleDeleteVehicle(vehicle.id)}
                        color="error"
                      >
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      <VehicleForm
        open={formOpen}
        onClose={closeForm}
        onSubmit={handleFormSubmit}
        vehicle={editingVehicle}
        title={editingVehicle ? 'Edit Vehicle' : 'Add New Vehicle'}
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

export default FleetManagement; 