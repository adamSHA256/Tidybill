-- Convert any stored 'overdue' status back to 'sent' (overdue is now computed)
UPDATE invoices SET status = 'sent' WHERE status = 'overdue';
