import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  TextInput,
  Select,
  NumberInput,
  Table,
  Button,
  Divider,
  ActionIcon,
  SimpleGrid,
  Loader,
  Center,
  Menu,
  Badge,
  Alert,
  Tooltip,
} from '@mantine/core'
import { DateInput } from '@mantine/dates'
import { IconTrash, IconPlus, IconPackage, IconAlertTriangle, IconInfoCircle } from '@tabler/icons-react'
import { formatMoney, type Item, type CustomerItem } from '../api/client'
import { useIsMobile } from '../hooks/useIsMobile'
import { useInvoiceForm } from '../hooks/useInvoiceForm'
import { InvoiceFormModals } from '../components/InvoiceFormModals'

export function InvoiceCreate() {
  const isMobile = useIsMobile()
  const form = useInvoiceForm()
  const { t } = form

  if (form.isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <div>
          <Title order={2}>{t('invoice.create_title')}</Title>
          <Text c="dimmed" size="sm">
            {form.duplicateFrom
              ? t('invoice.duplicate_subtitle').replace('{number}', form.duplicateFrom.invoice_number)
              : t('invoice.create_subtitle')}
          </Text>
        </div>
        <Group>
          <Button variant="default" onClick={() => form.navigate('/invoices')}>{t('common.cancel')}</Button>
          <Button onClick={form.handleCreate} loading={form.createPending}>{t('common.create')}</Button>
        </Group>
      </Group>

      <Paper p="md" radius="md" withBorder>
        <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
          <TextInput label={
            <Group gap={4}>
              <span>{t('invoice.invoice_number')}</span>
              <Tooltip label={t('invoice.invoice_number_hint')} multiline w={300} withArrow>
                <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
              </Tooltip>
            </Group>
          } placeholder={t('invoice.invoice_number_placeholder')} value={form.invoiceNumber} onChange={(e) => form.setInvoiceNumber(e.currentTarget.value)} />
          <DateInput label={t('invoice.issue_date')} valueFormat="DD.MM.YYYY" value={form.issueDate} onChange={form.setIssueDate} clearable />
          <DateInput
            label={
              <Group gap={4}>
                <span>{t('invoice.taxable_date')}</span>
                <Tooltip label={t('invoice.taxable_date_hint')} multiline w={300} withArrow>
                  <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
                </Tooltip>
              </Group>
            }
            valueFormat="DD.MM.YYYY"
            value={form.taxableDate ?? form.issueDate}
            onChange={form.setTaxableDate}
            clearable
          />
          <Select label={t('invoice.payment_method')} data={form.paymentTypeSelectData} value={form.paymentMethod} onChange={form.handlePaymentTypeSelect} searchable />
          <Select label={t('invoice.currency')} data={form.currencyData} value={form.currency} onChange={(v) => form.handleCurrencySelect(v, 'invoice')} searchable />
          <div>
            <DateInput
              label={t('invoice.due_date')}
              valueFormat="DD.MM.YYYY"
              value={form.dueDate}
              onChange={form.setDueDate}
              clearable
              styles={form.dueDateChangedByCustomer ? { input: { borderColor: 'var(--mantine-primary-color-6)', borderWidth: 2 } } : undefined}
            />
            {form.dueDateChangedByCustomer && (
              <Text size="xs" c="var(--mantine-primary-color-7)" mt={4}>{t('invoice.due_date_changed_by_customer')}</Text>
            )}
          </div>
          {form.requiresBankInfo && (
            <div>
              <TextInput
                label={t('invoice.variable_symbol_label')}
                value={form.variableSymbol}
                onChange={(e) => form.setVariableSymbol(e.currentTarget.value)}
                styles={form.vsChangedByInvoiceNumber ? { input: { borderColor: 'var(--mantine-primary-color-6)', borderWidth: 2 } } : undefined}
              />
              {form.vsChangedByInvoiceNumber && (
                <Text size="xs" c="var(--mantine-primary-color-7)" mt={4}>{t('invoice.vs_changed_by_invoice_number')}</Text>
              )}
            </div>
          )}
        </SimpleGrid>
        {form.currencyMismatch && (
          <Alert variant="light" color="orange" mt="sm" icon={<IconAlertTriangle size={16} />}>
            {t('invoice.currency_mismatch').replace('{bank}', form.selectedBank!.currency).replace('{invoice}', form.currency)}
          </Alert>
        )}
      </Paper>

      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('invoice.supplier_you')}</Text>
          {form.suppliers && form.suppliers.length === 0 ? (
            <Paper
              p="xl"
              radius="md"
              withBorder
              style={{ borderStyle: 'dashed', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 100 }}
              onClick={form.openSupplierModal}
            >
              <Stack align="center" gap="xs">
                <IconPlus size={24} color="gray" />
                <Text c="dimmed" size="sm">{t('invoice.no_suppliers_create')}</Text>
              </Stack>
            </Paper>
          ) : (
            <Select
              label={t('invoice.select_supplier')}
              placeholder={t('invoice.select_supplier_placeholder')}
              data={form.supplierData}
              value={form.supplierId}
              onChange={form.handleSupplierSelect}
            />
          )}
          {form.selectedSupplier && (
            <Stack gap={4} mt="sm">
              <Text size="sm" c="dimmed">
                ICO: {form.selectedSupplier.ico}{form.selectedSupplier.dic && ` | DIC: ${form.selectedSupplier.dic}`}{form.selectedSupplier.ic_dph && ` | IC DPH: ${form.selectedSupplier.ic_dph}`}
              </Text>
              <Text size="sm" c="dimmed">{form.selectedSupplier.street}, {form.selectedSupplier.city}, {form.selectedSupplier.zip}</Text>
            </Stack>
          )}
          {form.requiresBankInfo && (
            <>
              {form.supplierId && form.bankAccounts && form.bankAccounts.length > 0 ? (
                <Select
                  label={t('invoice.bank_account')}
                  mt="sm"
                  data={form.bankData}
                  value={form.bankAccountId}
                  onChange={form.handleBankSelect}
                />
              ) : form.supplierId && form.bankAccounts && form.bankAccounts.length === 0 ? (
                <Button variant="light" mt="sm" leftSection={<IconPlus size={14} />} onClick={form.openBankModal} fullWidth>
                  {t('invoice.create_first_bank_account')}
                </Button>
              ) : null}
            </>
          )}
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('invoice.customer')}</Text>
          <Select
            label={t('invoice.select_customer')}
            placeholder={t('invoice.search_customer')}
            data={form.customerData}
            value={form.customerId}
            onChange={form.handleCustomerSelect}
            searchable
          />
          {form.selectedCustomer && (
            <Stack gap={4} mt="sm">
              <Text size="sm" c="dimmed">
                ICO: {form.selectedCustomer.ico}{form.selectedCustomer.dic && ` | DIC: ${form.selectedCustomer.dic}`}{form.selectedCustomer.ic_dph && ` | IC DPH: ${form.selectedCustomer.ic_dph}`}
              </Text>
              <Text size="sm" c="dimmed">{form.selectedCustomer.street}, {form.selectedCustomer.city}, {form.selectedCustomer.zip}</Text>
            </Stack>
          )}
        </Paper>
      </SimpleGrid>

      <Paper p="md" radius="md" withBorder>
        <Group justify="space-between" mb="md">
          <Text fw={500}>{t('invoice.items')}</Text>
          <Group gap="xs">
            <Menu shadow="md" width={320} position="bottom-end">
              <Menu.Target>
                <Button size="xs" variant="light" leftSection={<IconPackage size={14} />}
                  disabled={!form.customerItems?.length && !form.globalSuggestions.length}>
                  {t('invoice.from_catalog')}
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                {(form.customerItems || []).length > 0 && (
                  <>
                    <Menu.Label>{t('invoice.customer_items')}</Menu.Label>
                    {(form.customerItems || []).map((ci: CustomerItem) => {
                      const catalogItem = form.mostUsedItems?.find((i: Item) => i.id === ci.item_id)
                      return (
                        <Menu.Item key={ci.id}
                          onClick={() => {
                            if (catalogItem) form.addFromCatalog(catalogItem, ci)
                            else form.addFromCatalog({
                              id: ci.item_id,
                              description: ci.item_description,
                              default_price: ci.last_price,
                              default_unit: ci.item_default_unit,
                              default_vat_rate: ci.item_default_vat,
                            } as Item, ci)
                          }}
                          rightSection={<Badge size="xs" variant="light">{ci.usage_count}x</Badge>}
                        >
                          <Text size="sm">{ci.item_description}</Text>
                          <Text size="xs" c="dimmed">{formatMoney(ci.last_price)} / {ci.item_default_unit}</Text>
                        </Menu.Item>
                      )
                    })}
                  </>
                )}
                {form.globalSuggestions.length > 0 && (
                  <>
                    <Menu.Label>{t('invoice.most_used')}</Menu.Label>
                    {form.globalSuggestions.slice(0, 10).map((item: Item) => (
                      <Menu.Item key={item.id}
                        onClick={() => form.addFromCatalog(item)}
                        rightSection={<Badge size="xs" variant="light" color="gray">{item.usage_count}x</Badge>}
                      >
                        <Text size="sm">{item.description}</Text>
                        <Text size="xs" c="dimmed">{formatMoney(item.default_price)} / {item.default_unit}</Text>
                      </Menu.Item>
                    ))}
                  </>
                )}
              </Menu.Dropdown>
            </Menu>
            <Button size="xs" leftSection={<IconPlus size={14} />} onClick={form.addItem}>{t('invoice.add_item')}</Button>
          </Group>
        </Group>

        <Table>
          <Table.Thead>
            <Table.Tr>
              <Table.Th style={{ width: '35%' }}>{t('invoice.description')}</Table.Th>
              <Table.Th style={{ width: '10%' }}>{t('invoice.qty')}</Table.Th>
              <Table.Th style={{ width: '10%' }}>{t('invoice.unit')}</Table.Th>
              <Table.Th style={{ width: '15%' }}>{t('invoice.unit_price')}</Table.Th>
              <Table.Th style={{ width: '10%' }}>{t('invoice.vat_pct')}</Table.Th>
              <Table.Th style={{ width: '15%' }}>{t('invoice.total_col')}</Table.Th>
              <Table.Th style={{ width: '5%' }}></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {form.items.map((item, i) => (
              <Table.Tr key={i}>
                <Table.Td>
                  <Group gap={4}>
                    <TextInput size="sm" placeholder={t('invoice.description_placeholder')} value={item.description}
                      onChange={(e) => form.updateItem(i, 'description', e.currentTarget.value)}
                      style={{ flex: 1 }} />
                    {item.item_id && <Badge size="xs" variant="light" color="blue">{t('invoice.catalog')}</Badge>}
                  </Group>
                </Table.Td>
                <Table.Td>
                  <NumberInput size="sm" min={1} value={item.quantity}
                    onChange={(val) => form.updateItem(i, 'quantity', val || 0)} />
                </Table.Td>
                <Table.Td>
                  <Select size="sm" data={form.unitSelectData} value={item.unit}
                    onChange={(val) => form.handleUnitSelect(val, i)} allowDeselect={false} />
                </Table.Td>
                <Table.Td>
                  <NumberInput size="sm" min={0} value={item.unit_price}
                    onChange={(val) => form.updateItem(i, 'unit_price', val || 0)} />
                </Table.Td>
                <Table.Td>
                  <Select size="sm" data={form.vatRateSelectData} value={String(item.vat_rate)}
                    onChange={(val) => form.handleVatRateSelect(val, i)} allowDeselect={false} />
                </Table.Td>
                <Table.Td>
                  <Text size="sm" fw={600}>{formatMoney(item.quantity * item.unit_price, form.currency)}</Text>
                </Table.Td>
                <Table.Td>
                  <ActionIcon color="red" variant="light" size="sm" onClick={() => form.removeItem(i)}
                    disabled={form.items.length === 1}>
                    <IconTrash size={14} />
                  </ActionIcon>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        <Divider my="md" />

        <Stack gap={4} align="end" pr="xl">
          <Group>
            <Text size="sm" c="dimmed" w={180} ta="right">{t('invoice.subtotal')}</Text>
            <Text size="sm" fw={600} w={120} ta="right">{formatMoney(form.subtotal, form.currency)}</Text>
          </Group>
          <Group>
            <Text size="sm" c="dimmed" w={180} ta="right">{t('invoice.vat')}</Text>
            <Text size="sm" fw={600} w={120} ta="right">{formatMoney(form.vatAmount, form.currency)}</Text>
          </Group>
          <Divider w={300} />
          <Group>
            <Text size="lg" fw={700} w={180} ta="right">{t('invoice.total')}</Text>
            <Text size="lg" fw={700} w={120} ta="right">{formatMoney(form.total, form.currency)}</Text>
          </Group>
        </Stack>
      </Paper>

      <TextInput label={t('invoice.notes_label')} placeholder={t('invoice.notes_placeholder')}
        value={form.notes} onChange={(e) => form.setNotes(e.currentTarget.value)} />

      <Group justify="end">
        <Button variant="default" onClick={() => form.navigate('/invoices')}>{t('common.cancel')}</Button>
        <Button onClick={form.handleCreate} loading={form.createPending}>{t('invoice.create_button')}</Button>
      </Group>

      <InvoiceFormModals modals={form.modals} isMobile={isMobile} t={t} />
    </Stack>
  )
}
