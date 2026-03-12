import { Modal, Stack, TextInput, SimpleGrid, Group, Button, Switch, Textarea, Select, NumberInput, Tooltip } from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import { CountrySelect } from './CountrySelect'
import { type InvoiceFormReturn } from '../hooks/useInvoiceForm'

interface Props {
  modals: InvoiceFormReturn['modals']
  isMobile: boolean | undefined
  t: InvoiceFormReturn['t']
}

export function InvoiceFormModals({ modals, isMobile, t }: Props) {
  const {
    // Supplier modal
    supplierModalOpen, setSupplierModalOpen,
    sName, setSName, sIco, setSIco, sDic, setSDic, sIcDph, setSIcDph,
    sStreet, setSStreet, sCity, setSCity, sZip, setSZip, sCountry, setSCountry,
    sEmail, setSEmail, sPhone, setSPhone, sWebsite, setSWebsite,
    sInvoicePrefix, setSInvoicePrefix, sIsVatPayer, setSIsVatPayer, sNotes, setSNotes,
    handleSaveSupplier, createSupplierPending,

    // Customer modal
    customerModalOpen, setCustomerModalOpen,
    cName, setCName, cIco, setCIco, cDic, setCDic, cIcDph, setCIcDph,
    cStreet, setCStreet, cCity, setCCity, cZip, setCZip, cCountry, setCCountry,
    cEmail, setCEmail, cPhone, setCPhone, cDueDays, setCDueDays, cNotes, setCNotes,
    handleSaveCustomer, createCustomerPending,

    // Bank modal
    bankModalOpen, setBankModalOpen,
    bName, setBName, bAccountNumber, setBAccountNumber, bIban, setBIban,
    bSwift, setBSwift, bCurrency, bIsDefault, setBIsDefault,
    bQrType, setBQrType,
    handleSaveBank, createBankPending,

    // Currency modal
    currencyModalOpen, setCurrencyModalOpen,
    newCurrencyCode, setNewCurrencyCode,
    handleAddCurrency,

    // VAT rate modal
    vatRateModalOpen, setVatRateModalOpen,
    newVatRateValue, setNewVatRateValue,
    handleAddVatRate,

    // Unit modal
    unitModalOpen, setUnitModalOpen,
    newUnitValue, setNewUnitValue,
    handleAddUnit,

    // Payment type modal
    paymentTypeModalOpen, setPaymentTypeModalOpen,
    newPaymentTypeName, setNewPaymentTypeName,
    handleAddPaymentType,

    // Shared data for modals
    currencyData,
    handleCurrencySelect,
  } = modals

  return (
    <>
      {/* Supplier creation modal */}
      <Modal opened={supplierModalOpen} onClose={() => setSupplierModalOpen(false)}
        title={t('supplier.new_title')} size="lg" fullScreen={isMobile}>
        <Stack gap="md">
          <TextInput label={t('supplier.name_label')} value={sName}
            onChange={(e) => setSName(e.currentTarget.value)} required />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('supplier.ico_label')} value={sIco}
              onChange={(e) => setSIco(e.currentTarget.value)} />
            <TextInput label={t('supplier.dic_label')} value={sDic}
              onChange={(e) => setSDic(e.currentTarget.value)} />
          </SimpleGrid>
          {sCountry.toUpperCase() === 'SK' && (
            <TextInput label={t('supplier.ic_dph_label')} value={sIcDph}
              onChange={(e) => setSIcDph(e.currentTarget.value)} />
          )}
          <TextInput label={t('supplier.street_label')} value={sStreet}
            onChange={(e) => setSStreet(e.currentTarget.value)} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('supplier.city_label')} value={sCity}
              onChange={(e) => setSCity(e.currentTarget.value)} />
            <TextInput label={t('supplier.zip_label')} value={sZip}
              onChange={(e) => setSZip(e.currentTarget.value)} />
          </SimpleGrid>
          <CountrySelect label={t('supplier.country_label')}
            value={sCountry} onChange={(v) => setSCountry(v || 'CZ')} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('supplier.email_label')} value={sEmail}
              onChange={(e) => setSEmail(e.currentTarget.value)} />
            <TextInput label={t('supplier.phone_label')} value={sPhone}
              onChange={(e) => setSPhone(e.currentTarget.value)} />
          </SimpleGrid>
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('supplier.website_label')} value={sWebsite}
              onChange={(e) => setSWebsite(e.currentTarget.value)} />
            <TextInput label={t('supplier.invoice_prefix_label')} value={sInvoicePrefix}
              onChange={(e) => setSInvoicePrefix(e.currentTarget.value)} />
          </SimpleGrid>
          <Switch label={t('supplier.is_vat_payer_label')} checked={sIsVatPayer}
            onChange={(e) => setSIsVatPayer(e.currentTarget.checked)} />
          <Textarea label={t('supplier.notes_label')} value={sNotes}
            onChange={(e) => setSNotes(e.currentTarget.value)} minRows={2} />
          <Group justify="end" mt="md">
            <Button variant="default" onClick={() => setSupplierModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleSaveSupplier} loading={createSupplierPending}>{t('common.create')}</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Customer creation modal */}
      <Modal opened={customerModalOpen} onClose={() => setCustomerModalOpen(false)}
        title={t('customer.new_title')} size="lg" fullScreen={isMobile}>
        <Stack gap="md">
          <TextInput label={t('customer.name_label')} value={cName}
            onChange={(e) => setCName(e.currentTarget.value)} required />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('customer.ico_label')} value={cIco}
              onChange={(e) => setCIco(e.currentTarget.value)} />
            <TextInput label={t('customer.dic_label')} value={cDic}
              onChange={(e) => setCDic(e.currentTarget.value)} />
          </SimpleGrid>
          {cCountry.toUpperCase() === 'SK' && (
            <TextInput label={t('customer.ic_dph_label')} value={cIcDph}
              onChange={(e) => setCIcDph(e.currentTarget.value)} />
          )}
          <TextInput label={t('customer.street_label')} value={cStreet}
            onChange={(e) => setCStreet(e.currentTarget.value)} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('customer.city_label')} value={cCity}
              onChange={(e) => setCCity(e.currentTarget.value)} />
            <TextInput label={t('customer.zip_label')} value={cZip}
              onChange={(e) => setCZip(e.currentTarget.value)} />
          </SimpleGrid>
          <CountrySelect label={t('customer.country_label')}
            value={cCountry} onChange={(v) => setCCountry(v || 'CZ')} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('customer.email_label')} value={cEmail}
              onChange={(e) => setCEmail(e.currentTarget.value)} />
            <TextInput label={t('customer.phone_label')} value={cPhone}
              onChange={(e) => setCPhone(e.currentTarget.value)} />
          </SimpleGrid>
          <NumberInput label={t('customer.default_due_days_label')}
            description={t('customer.default_due_days_desc')}
            value={cDueDays} onChange={(v) => setCDueDays(Number(v) || 0)}
            min={0} max={365} w={200} />
          <Textarea label={
            <Group gap={4}>
              <span>{t('customer.notes_label')}</span>
              <Tooltip label={t('customer.notes_hint')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
                <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
              </Tooltip>
            </Group>
          } value={cNotes}
            onChange={(e) => setCNotes(e.currentTarget.value)} minRows={2} />
          <Group justify="end" mt="md">
            <Button variant="default" onClick={() => setCustomerModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleSaveCustomer} loading={createCustomerPending}>{t('common.create')}</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Bank account creation modal */}
      <Modal opened={bankModalOpen} onClose={() => setBankModalOpen(false)}
        title={t('bank_account.new_title')} size="md" fullScreen={isMobile}>
        <Stack gap="md">
          <TextInput label={t('bank_account.name_label')} value={bName}
            onChange={(e) => setBName(e.currentTarget.value)} />
          <TextInput label={t('bank_account.account_number_label')} value={bAccountNumber}
            onChange={(e) => setBAccountNumber(e.currentTarget.value)} required />
          <TextInput label={t('bank_account.iban_label')} value={bIban}
            onChange={(e) => setBIban(e.currentTarget.value)} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('bank_account.swift_label')} value={bSwift}
              onChange={(e) => setBSwift(e.currentTarget.value)} />
            <Select label={t('bank_account.currency_label')} data={currencyData}
              value={bCurrency} onChange={(v) => handleCurrencySelect(v, 'bank')} searchable />
          </SimpleGrid>
          <Select label={
            <Group gap={4}>
              <span>{t('bank_account.qr_type_label')}</span>
              <Tooltip label={t('bank_account.qr_type_hint')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
                <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
              </Tooltip>
            </Group>
          }
            data={[
              { value: 'spayd', label: t('bank_account.qr_spayd') },
              { value: 'pay_by_square', label: t('bank_account.qr_pbs') },
              { value: 'epc', label: t('bank_account.qr_epc') },
              { value: 'none', label: t('bank_account.qr_none') },
            ]}
            value={bQrType} onChange={(v) => setBQrType(v || 'spayd')}
            allowDeselect={false} />
          <Switch label={
            <Group gap={4}>
              <span>{t('bank_account.is_default_label')}</span>
              <Tooltip label={t('bank_account.is_default_hint')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
                <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
              </Tooltip>
            </Group>
          } checked={bIsDefault}
            onChange={(e) => setBIsDefault(e.currentTarget.checked)} />
          <Group justify="end" mt="md">
            <Button variant="default" onClick={() => setBankModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleSaveBank} loading={createBankPending}>{t('common.create')}</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Add currency modal */}
      <Modal opened={currencyModalOpen} onClose={() => setCurrencyModalOpen(false)}
        title={t('invoice.add_currency')} size="xs">
        <Stack gap="md">
          <TextInput label={t('invoice.currency_code')} placeholder="BTC"
            value={newCurrencyCode} onChange={(e) => setNewCurrencyCode(e.currentTarget.value.toUpperCase())}
            maxLength={10}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddCurrency() }}
          />
          <Group justify="end">
            <Button variant="default" onClick={() => setCurrencyModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleAddCurrency} disabled={!newCurrencyCode.trim()}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Add VAT rate modal */}
      <Modal opened={vatRateModalOpen} onClose={() => setVatRateModalOpen(false)}
        title={t('invoice.new_vat_rate')} size="xs">
        <Stack gap="md">
          <TextInput label={t('invoice.new_vat_rate_label')} placeholder="15"
            value={newVatRateValue} onChange={(e) => setNewVatRateValue(e.currentTarget.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddVatRate() }}
          />
          <Group justify="end">
            <Button variant="default" onClick={() => setVatRateModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleAddVatRate} disabled={!newVatRateValue.trim()}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Add unit modal */}
      <Modal opened={unitModalOpen} onClose={() => setUnitModalOpen(false)}
        title={t('invoice.new_unit')} size="xs">
        <Stack gap="md">
          <TextInput label={t('invoice.new_unit_label')} placeholder="bal"
            value={newUnitValue} onChange={(e) => setNewUnitValue(e.currentTarget.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddUnit() }}
          />
          <Group justify="end">
            <Button variant="default" onClick={() => setUnitModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleAddUnit} disabled={!newUnitValue.trim()}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Add payment type modal */}
      <Modal opened={paymentTypeModalOpen} onClose={() => setPaymentTypeModalOpen(false)}
        title={t('settings.payment_type_placeholder')} size="xs">
        <Stack gap="md">
          <TextInput label={t('invoice.payment_method')} placeholder={t('settings.payment_type_placeholder')}
            value={newPaymentTypeName} onChange={(e) => setNewPaymentTypeName(e.currentTarget.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddPaymentType() }}
          />
          <Group justify="end">
            <Button variant="default" onClick={() => setPaymentTypeModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleAddPaymentType} disabled={!newPaymentTypeName.trim()}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>
    </>
  )
}
