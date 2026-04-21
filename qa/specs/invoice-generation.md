---
title: Invoice Generation
summary: Generate PDF invoices from billing data with line items and tax calculations
type: functional
---

## Overview

The billing module must generate downloadable PDF invoices for completed orders.

## Requirements

- Pull line items from the orders table
- Calculate subtotal, tax, and total
- Apply regional tax rules (GST, VAT, sales tax)
- Generate PDF with company logo, customer details, and itemized breakdown
- Store generated PDF in object storage with a unique URL
- Send invoice via email to the customer

## Acceptance Criteria

- Invoice PDF matches the approved template
- Tax calculations are correct for AU, US, and EU regions
- PDF is accessible via a signed URL for 30 days
- Email delivery succeeds with the PDF attachment
