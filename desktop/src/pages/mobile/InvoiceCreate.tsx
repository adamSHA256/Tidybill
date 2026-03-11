import {
  Stack,
  Group,
  Paper,
  Text,
  Title,
  TextInput,
  Select,
  NumberInput,
  Button,
  ActionIcon,
  Divider,
  Badge,
  Menu,
  Loader,
  Center,
  Alert,
  Tooltip,
  SimpleGrid,
} from '@mantine/core'
import { DateInput } from '@mantine/dates'
import {
  IconTrash,
  IconPlus,
  IconPackage,
  IconAlertTriangle,
  IconInfoCircle,
  IconChevronDown,
} from '@tabler/icons-react'
import { useInvoiceForm } from '../../hooks/useInvoiceForm'
import { InvoiceFormModals } from '../../components/InvoiceFormModals'
import { useIsMobile } from '../../hooks/useIsMobile'
import { formatMoney, type Item, type CustomerItem } from '../../api/client'

export function MobileInvoiceCreate() {
  const form = useInvoiceForm()
  const isMobile = useIsMobile()

  const {
    navigate,
    t,
    duplicateFrom,
    isLoading,
    // Core state
    issueDate,
    setIssueDate,
    taxableDate,
    setTaxableDate,
    dueDate,
    setDueDate,
    currency,
    invoiceNumber,
    setInvoiceNumber,
    paymentMethod,
    notes,
    setNotes,
    variableSymbol,
    setVariableSymbol,
    items,
    // UI indicators
    dueDateChangedByCustomer,
    vsChangedByInvoiceNumber,
    currencyMismatch,
    // Entities
    suppliers,
    selectedSupplier,
    selectedCustomer,
    selectedBank,
    bankAccounts,
    requiresBankInfo,
    supplierId: selectedSupplierId,
    customerId,
    bankAccountId: selectedBankId,
    // Computed
    subtotal,
    vatAmount,
    total,
    // Select data
    supplierData,
    customerData,
    bankData,
    currencyData,
    vatRateSelectData,
    unitSelectData,
    paymentTypeSelectData,
    // Catalog
    customerItems,
    globalSuggestions,
    mostUsedItems,
    // Create
    handleCreate,
    createPending,
    // Handlers
    handleSupplierSelect,
    handleCustomerSelect,
    handleBankSelect,
    handleCurrencySelect,
    handleVatRateSelect,
    handleUnitSelect,
    handlePaymentTypeSelect,
    // Item handlers
    addItem,
    removeItem,
    updateItem,
    addFromCatalog,
    // Modal openers
    openSupplierModal,
    openBankModal,
  } = form

  if (isLoading) return <Center h={300}><Loader /></Center>

  return (
    <Stack gap="md">
      {/* Header */}
      <Group justify="space-between">
        <div>
          <Title order={3}>{t('invoice.create_title')}</Title>
          {duplicateFrom && (
            <Text c="dimmed" size="xs">
              {t('invoice.duplicate_subtitle').replace('{number}', duplicateFrom.invoice_number)}
            </Text>
          )}
        </div>
        <Button size="sm" onClick={handleCreate} loading={createPending}>
          {t('common.create')}
        </Button>
      </Group>

      {/* Invoice details */}
      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="sm">{t('invoice.invoice_number')}</Text>
        <Stack gap="sm">
          <TextInput
            label={
              <Group gap={4}>
                <span>{t('invoice.invoice_number')}</span>
                <Tooltip label={t('invoice.invoice_number_hint')} multiline w={250} withArrow>
                  <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
                </Tooltip>
              </Group>
            }
            placeholder={t('invoice.invoice_number_placeholder')}
            value={invoiceNumber}
            onChange={(e) => setInvoiceNumber(e.currentTarget.value)}
          />
          <DateInput
            label={t('invoice.issue_date')}
            valueFormat="DD.MM.YYYY"
            value={issueDate}
            onChange={setIssueDate}
            clearable
          />
          <DateInput
            label={
              <Group gap={4}>
                <span>{t('invoice.taxable_date')}</span>
                <Tooltip label={t('invoice.taxable_date_hint')} multiline w={250} withArrow>
                  <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
                </Tooltip>
              </Group>
            }
            valueFormat="DD.MM.YYYY"
            value={taxableDate ?? issueDate}
            onChange={setTaxableDate}
            clearable
          />
          <Select
            label={t('invoice.payment_method')}
            data={paymentTypeSelectData}
            value={paymentMethod}
            onChange={handlePaymentTypeSelect}
            searchable
          />
          <Select
            label={t('invoice.currency')}
            data={currencyData}
            value={currency}
            onChange={(v) => handleCurrencySelect(v, 'invoice')}
            searchable
          />
          <div>
            <DateInput
              label={t('invoice.due_date')}
              valueFormat="DD.MM.YYYY"
              value={dueDate}
              onChange={setDueDate}
              clearable
              styles={
                dueDateChangedByCustomer
                  ? { input: { borderColor: 'var(--mantine-primary-color-6)', borderWidth: 2 } }
                  : undefined
              }
            />
            {dueDateChangedByCustomer && (
              <Text size="xs" c="var(--mantine-primary-color-7)" mt={4}>
                {t('invoice.due_date_changed_by_customer')}
              </Text>
            )}
          </div>
          {requiresBankInfo && (
            <div>
              <TextInput
                label={t('invoice.variable_symbol_label')}
                value={variableSymbol}
                onChange={(e) => setVariableSymbol(e.currentTarget.value)}
                styles={
                  vsChangedByInvoiceNumber
                    ? { input: { borderColor: 'var(--mantine-primary-color-6)', borderWidth: 2 } }
                    : undefined
                }
              />
              {vsChangedByInvoiceNumber && (
                <Text size="xs" c="var(--mantine-primary-color-7)" mt={4}>
                  {t('invoice.vs_changed_by_invoice_number')}
                </Text>
              )}
            </div>
          )}
        </Stack>
        {currencyMismatch && (
          <Alert variant="light" color="orange" mt="sm" icon={<IconAlertTriangle size={16} />}>
            {t('invoice.currency_mismatch')
              .replace('{bank}', selectedBank!.currency)
              .replace('{invoice}', currency)}
          </Alert>
        )}
      </Paper>

      {/* Supplier section */}
      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="sm">{t('invoice.supplier_you')}</Text>
        {suppliers && suppliers.length === 0 ? (
          <Paper
            p="xl"
            radius="md"
            withBorder
            style={{
              borderStyle: 'dashed',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              minHeight: 80,
            }}
            onClick={openSupplierModal}
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
            data={supplierData}
            value={selectedSupplierId}
            onChange={handleSupplierSelect}
          />
        )}
        {selectedSupplier && (
          <Stack gap={4} mt="sm">
            <Text size="sm" c="dimmed">
              ICO: {selectedSupplier.ico}
              {selectedSupplier.dic && ` | DIC: ${selectedSupplier.dic}`}
              {selectedSupplier.ic_dph && ` | IC DPH: ${selectedSupplier.ic_dph}`}
            </Text>
            <Text size="sm" c="dimmed">
              {selectedSupplier.street}, {selectedSupplier.city}, {selectedSupplier.zip}
            </Text>
          </Stack>
        )}
        {requiresBankInfo && (
          <>
            {selectedSupplierId && bankAccounts && bankAccounts.length > 0 ? (
              <Select
                label={t('invoice.bank_account')}
                mt="sm"
                data={bankData}
                value={selectedBankId}
                onChange={handleBankSelect}
              />
            ) : selectedSupplierId && bankAccounts && bankAccounts.length === 0 ? (
              <Button
                variant="light"
                mt="sm"
                leftSection={<IconPlus size={14} />}
                onClick={openBankModal}
                fullWidth
              >
                {t('invoice.create_first_bank_account')}
              </Button>
            ) : null}
          </>
        )}
      </Paper>

      {/* Customer section */}
      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="sm">{t('invoice.customer')}</Text>
        <Select
          label={t('invoice.select_customer')}
          placeholder={t('invoice.search_customer')}
          data={customerData}
          value={customerId}
          onChange={handleCustomerSelect}
          searchable
        />
        {selectedCustomer && (
          <Stack gap={4} mt="sm">
            <Text size="sm" c="dimmed">
              ICO: {selectedCustomer.ico}
              {selectedCustomer.dic && ` | DIC: ${selectedCustomer.dic}`}
              {selectedCustomer.ic_dph && ` | IC DPH: ${selectedCustomer.ic_dph}`}
            </Text>
            <Text size="sm" c="dimmed">
              {selectedCustomer.street}, {selectedCustomer.city}, {selectedCustomer.zip}
            </Text>
          </Stack>
        )}
      </Paper>

      {/* Items section */}
      <Paper p="md" radius="md" withBorder>
        <Group justify="space-between" mb="md">
          <Text fw={500}>{t('invoice.items')}</Text>
        </Group>

        {/* Catalog menu + Add item button */}
        <Group gap="xs" mb="md">
          <Menu shadow="md" width={280} position="bottom-start">
            <Menu.Target>
              <Button
                size="xs"
                variant="light"
                leftSection={<IconPackage size={14} />}
                rightSection={<IconChevronDown size={14} />}
                disabled={!customerItems?.length && !globalSuggestions.length}
              >
                {t('invoice.from_catalog')}
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              {(customerItems || []).length > 0 && (
                <>
                  <Menu.Label>{t('invoice.customer_items')}</Menu.Label>
                  {(customerItems || []).map((ci: CustomerItem) => {
                    const catalogItem = mostUsedItems?.find((i: Item) => i.id === ci.item_id)
                    return (
                      <Menu.Item
                        key={ci.id}
                        onClick={() => {
                          if (catalogItem) addFromCatalog(catalogItem, ci)
                          else
                            addFromCatalog(
                              {
                                id: ci.item_id,
                                description: ci.item_description,
                                default_price: ci.last_price,
                                default_unit: ci.item_default_unit,
                                default_vat_rate: ci.item_default_vat,
                              } as Item,
                              ci
                            )
                        }}
                        rightSection={
                          <Badge size="xs" variant="light">
                            {ci.usage_count}x
                          </Badge>
                        }
                      >
                        <Text size="sm">{ci.item_description}</Text>
                        <Text size="xs" c="dimmed">
                          {formatMoney(ci.last_price)} / {ci.item_default_unit}
                        </Text>
                      </Menu.Item>
                    )
                  })}
                </>
              )}
              {globalSuggestions.length > 0 && (
                <>
                  <Menu.Label>{t('invoice.most_used')}</Menu.Label>
                  {globalSuggestions.slice(0, 10).map((item: Item) => (
                    <Menu.Item
                      key={item.id}
                      onClick={() => addFromCatalog(item)}
                      rightSection={
                        <Badge size="xs" variant="light" color="gray">
                          {item.usage_count}x
                        </Badge>
                      }
                    >
                      <Text size="sm">{item.description}</Text>
                      <Text size="xs" c="dimmed">
                        {formatMoney(item.default_price)} / {item.default_unit}
                      </Text>
                    </Menu.Item>
                  ))}
                </>
              )}
            </Menu.Dropdown>
          </Menu>
          <Button size="xs" leftSection={<IconPlus size={14} />} onClick={addItem}>
            {t('invoice.add_item')}
          </Button>
        </Group>

        {/* Item cards */}
        <Stack gap="sm">
          {items.map((item, i) => (
            <Paper key={i} p="sm" radius="sm" withBorder>
              <Group justify="space-between" mb="xs">
                <Group gap="xs">
                  <Text size="sm" fw={500}>
                    {t('invoice.items')} {i + 1}
                  </Text>
                  {item.item_id && (
                    <Badge size="xs" variant="light" color="blue">
                      {t('invoice.catalog')}
                    </Badge>
                  )}
                </Group>
                <ActionIcon
                  color="red"
                  variant="light"
                  size="sm"
                  onClick={() => removeItem(i)}
                  disabled={items.length === 1}
                >
                  <IconTrash size={14} />
                </ActionIcon>
              </Group>

              <TextInput
                size="sm"
                label={t('invoice.description')}
                placeholder={t('invoice.description_placeholder')}
                value={item.description}
                onChange={(e) => updateItem(i, 'description', e.currentTarget.value)}
                mb="xs"
              />

              <SimpleGrid cols={2} spacing="xs" mb="xs">
                <NumberInput
                  size="sm"
                  label={t('invoice.qty')}
                  min={1}
                  value={item.quantity}
                  onChange={(val) => updateItem(i, 'quantity', val || 0)}
                />
                <Select
                  size="sm"
                  label={t('invoice.unit')}
                  data={unitSelectData}
                  value={item.unit}
                  onChange={(val) => handleUnitSelect(val, i)}
                  allowDeselect={false}
                />
              </SimpleGrid>

              <SimpleGrid cols={2} spacing="xs" mb="xs">
                <NumberInput
                  size="sm"
                  label={t('invoice.unit_price')}
                  min={0}
                  value={item.unit_price}
                  onChange={(val) => updateItem(i, 'unit_price', val || 0)}
                />
                <Select
                  size="sm"
                  label={t('invoice.vat_pct')}
                  data={vatRateSelectData}
                  value={String(item.vat_rate)}
                  onChange={(val) => handleVatRateSelect(val, i)}
                  allowDeselect={false}
                />
              </SimpleGrid>

              <Text size="sm" fw={600} ta="right">
                {t('invoice.total_col')}: {formatMoney(item.quantity * item.unit_price, currency)}
              </Text>
            </Paper>
          ))}
        </Stack>

        {/* Totals */}
        <Divider my="md" />
        <Stack gap={4} align="center">
          <Group>
            <Text size="sm" c="dimmed" w={140} ta="right">
              {t('invoice.subtotal')}
            </Text>
            <Text size="sm" fw={600} w={120} ta="right">
              {formatMoney(subtotal, currency)}
            </Text>
          </Group>
          <Group>
            <Text size="sm" c="dimmed" w={140} ta="right">
              {t('invoice.vat')}
            </Text>
            <Text size="sm" fw={600} w={120} ta="right">
              {formatMoney(vatAmount, currency)}
            </Text>
          </Group>
          <Divider w={260} />
          <Group>
            <Text size="lg" fw={700} w={140} ta="right">
              {t('invoice.total')}
            </Text>
            <Text size="lg" fw={700} w={120} ta="right">
              {formatMoney(total, currency)}
            </Text>
          </Group>
        </Stack>
      </Paper>

      {/* Notes */}
      <TextInput
        label={t('invoice.notes_label')}
        placeholder={t('invoice.notes_placeholder')}
        value={notes}
        onChange={(e) => setNotes(e.currentTarget.value)}
      />

      {/* Bottom action bar */}
      <Group justify="space-between" pb="md">
        <Button variant="default" onClick={() => navigate('/invoices')}>
          {t('common.cancel')}
        </Button>
        <Button onClick={handleCreate} loading={createPending}>
          {t('invoice.create_button')}
        </Button>
      </Group>

      {/* Shared modals */}
      <InvoiceFormModals modals={form.modals} isMobile={isMobile} t={form.t} />
    </Stack>
  )
}
