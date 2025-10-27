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
import CostForm from './CostForm';
import apiService from '../services/api';
import { Cost } from '../types';

interface CostManagementProps {
  timeRange?: { from?: string; to?: string };
}

const CostManagement: React.FC<CostManagementProps> = ({ timeRange }) => {
  const [costs, setCosts] = useState<Cost[]>([]);
  const [loading, setLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error';
  }>({ open: false, message: '', severity: 'success' });

  const loadCosts = async () => {
    setLoading(true);
    try {
      const data = await apiService.getCosts(timeRange);
      setCosts(data);
    } catch (error) {
      console.error('Error loading costs:', error);
      setSnackbar({
        open: true,
        message: 'Failed to load cost data',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCosts();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [timeRange?.from, timeRange?.to]);

  const handleAddCost = async (costData: Omit<Cost, 'id'>) => {
    try {
      await apiService.postCost(costData);
      setSnackbar({
        open: true,
        message: 'Cost record created successfully',
        severity: 'success',
      });
      loadCosts();
    } catch (error) {
      console.error('Error creating cost:', error);
      setSnackbar({
        open: true,
        message: 'Failed to create cost record',
        severity: 'error',
      });
    }
  };

  const handleDeleteCost = async (id: string) => {
    const ok = window.confirm('Delete this cost record?');
    if (!ok) return;
    try {
      await apiService.deleteCost(id);
      setSnackbar({ open: true, message: 'Cost deleted', severity: 'success' });
      loadCosts();
    } catch (error) {
      console.error('Error deleting cost:', error);
      setSnackbar({ open: true, message: 'Failed to delete cost', severity: 'error' });
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
      case 'paid': return 'success';
      case 'pending': return 'warning';
      case 'disputed': return 'error';
      case 'cancelled': return 'default';
      default: return 'default';
    }
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'fuel': return 'primary';
      case 'maintenance': return 'secondary';
      case 'insurance': return 'info';
      case 'registration': return 'warning';
      case 'tolls': return 'error';
      case 'parking': return 'default';
      case 'other': return 'default';
      default: return 'default';
    }
  };

  const getPaymentMethodLabel = (method: string) => {
    return method.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const totalCosts = costs.reduce((sum, cost) => sum + cost.amount, 0);

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5" component="h2">
          Cost Management
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={openAddForm}
        >
          Add Cost
        </Button>
      </Box>

      <Paper sx={{ p: 2, mb: 2 }}>
        <Typography variant="h6" gutterBottom>
          Total Costs: ${totalCosts.toFixed(2)}
        </Typography>
      </Paper>

      <Paper sx={{ p: 2 }}>
        {loading ? (
          <Typography>Loading cost data...</Typography>
        ) : costs.length === 0 ? (
          <Typography color="text.secondary" align="center" py={4}>
            No cost records found. Add your first cost record to get started.
          </Typography>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Vehicle ID</TableCell>
                  <TableCell>Category</TableCell>
                  <TableCell>Description</TableCell>
                  <TableCell>Amount ($)</TableCell>
                  <TableCell>Date</TableCell>
                  <TableCell>Vendor</TableCell>
                  <TableCell>Payment Method</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {costs.map((cost) => (
                  <TableRow key={cost.id}>
                    <TableCell>{cost.vehicle_id}</TableCell>
                    <TableCell>
                      <Chip
                        label={cost.category}
                        color={getCategoryColor(cost.category) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>{cost.description}</TableCell>
                    <TableCell>${cost.amount.toFixed(2)}</TableCell>
                    <TableCell>{formatDate(cost.date)}</TableCell>
                    <TableCell>{cost.vendor}</TableCell>
                    <TableCell>{getPaymentMethodLabel(cost.payment_method)}</TableCell>
                    <TableCell>
                      <Chip
                        label={cost.status}
                        color={getStatusColor(cost.status) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Tooltip title="Delete">
                        <IconButton size="small" onClick={() => handleDeleteCost(cost.id!)}>
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

      <CostForm
        open={formOpen}
        onClose={closeForm}
        onSubmit={handleAddCost}
        title="Add Cost Record"
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

export default CostManagement; 