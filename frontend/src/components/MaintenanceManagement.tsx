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
import MaintenanceForm from './MaintenanceForm';
import apiService from '../services/api';
import { Maintenance } from '../types';

interface MaintenanceManagementProps {
  timeRange?: { from?: string; to?: string };
}

const MaintenanceManagement: React.FC<MaintenanceManagementProps> = ({ timeRange }) => {
  const [maintenance, setMaintenance] = useState<Maintenance[]>([]);
  const [loading, setLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error';
  }>({ open: false, message: '', severity: 'success' });

  const loadMaintenance = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiService.getMaintenance(timeRange);
      setMaintenance(data);
    } catch (error) {
      console.error('Error loading maintenance:', error);
      setSnackbar({
        open: true,
        message: 'Failed to load maintenance data',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    loadMaintenance();
  }, [loadMaintenance]);

  const handleAddMaintenance = async (maintenanceData: Omit<Maintenance, 'id'>) => {
    try {
      await apiService.postMaintenance(maintenanceData);
      setSnackbar({
        open: true,
        message: 'Maintenance record created successfully',
        severity: 'success',
      });
      loadMaintenance();
    } catch (error) {
      console.error('Error creating maintenance:', error);
      setSnackbar({
        open: true,
        message: 'Failed to create maintenance record',
        severity: 'error',
      });
    }
  };

  const handleDeleteMaintenance = async (id: string) => {
    const ok = window.confirm('Delete this maintenance record?');
    if (!ok) return;
    try {
      await apiService.deleteMaintenance(id);
      setSnackbar({ open: true, message: 'Maintenance deleted', severity: 'success' });
      loadMaintenance();
    } catch (error) {
      console.error('Error deleting maintenance:', error);
      setSnackbar({ open: true, message: 'Failed to delete maintenance', severity: 'error' });
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
      case 'scheduled': return 'info';
      case 'cancelled': return 'error';
      default: return 'default';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'critical': return 'error';
      case 'high': return 'warning';
      case 'medium': return 'info';
      case 'low': return 'success';
      default: return 'default';
    }
  };

  const getServiceTypeLabel = (serviceType: string) => {
    return serviceType.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5" component="h2">
          Maintenance Management
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={openAddForm}
        >
          Add Maintenance
        </Button>
      </Box>

      <Paper sx={{ p: 2 }}>
        {loading ? (
          <Typography>Loading maintenance data...</Typography>
        ) : maintenance.length === 0 ? (
          <Typography color="text.secondary" align="center" py={4}>
            No maintenance records found. Add your first maintenance record to get started.
          </Typography>
        ) : (
          <TableContainer sx={{ overflowX: 'auto' }}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Vehicle ID</TableCell>
                  <TableCell>Service Type</TableCell>
                  <TableCell>Service Date</TableCell>
                  <TableCell>Next Service</TableCell>
                  <TableCell>Mileage (km)</TableCell>
                  <TableCell>Cost ($)</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Priority</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {maintenance.map((record) => (
                  <TableRow key={record.id}>
                    <TableCell>{record.vehicle_id}</TableCell>
                    <TableCell>{getServiceTypeLabel(record.service_type)}</TableCell>
                    <TableCell>{formatDate(record.service_date)}</TableCell>
                    <TableCell>{formatDate(record.next_service_date)}</TableCell>
                    <TableCell>{record.mileage.toFixed(0)}</TableCell>
                    <TableCell>${record.cost.toFixed(2)}</TableCell>
                    <TableCell>
                      <Chip
                        label={record.status.replace('_', ' ')}
                        color={getStatusColor(record.status) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={record.priority}
                        color={getPriorityColor(record.priority) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Tooltip title="Delete">
                        <IconButton size="small" onClick={() => handleDeleteMaintenance(record.id!)}>
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

      <MaintenanceForm
        open={formOpen}
        onClose={closeForm}
        onSubmit={handleAddMaintenance}
        title="Add Maintenance Record"
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

export default MaintenanceManagement; 