import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Badge,
  Button,
  Table,
  TextInput,
  Textarea,
  Select,
  Switch,
  Modal,
  Loader,
  Center,
  Avatar,
  FileButton,
  ActionIcon,
  Divider,
  Box,
  SegmentedControl,
  Tooltip,
  SimpleGrid,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconUpload, IconTrash, IconPencil, IconBuildingBank, IconInfoCircle, IconAt } from '@tabler/icons-react'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type Supplier, type BankAccount } from '../api/client'
import { CountrySelect } from '../components/CountrySelect'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'
import { SmtpConfigForm } from '../components/SmtpConfigForm'

function BankAccountsRow({ supplierId, supplierName, onEdit, onDelete, onCreate }: {
  supplierId: string
  supplierName: string
  onEdit: (ba: BankAccount) => void
  onDelete: (ba: BankAccount) => void
  onCreate: (supplierId: string) => void
}) {
  const { t } = useT()
  const { data: bankAccounts, isLoading } = useQuery({
    queryKey: ['bank-accounts', supplierId],
    queryFn: () => api.getBankAccounts(supplierId),
  })

  return (
    <Table.Tr>
      <Table.Td colSpan={7} p={0}>
        <Box px="lg" py="md" bg="var(--mantine-color-default-hover)">
          <Group justify="space-between" mb="sm">
            <Group gap="xs">
              <IconBuildingBank size={16} />
              <Text size="sm" fw={600}>{supplierName}</Text>
            </Group>
            <Button size="xs" variant="light" leftSection={<IconPlus size={14} />}
              onClick={() => onCreate(supplierId)}>
              {t('bank_account.add')}
            </Button>
          </Group>
          {isLoading ? (
            <Center py="xs"><Loader size="xs" /></Center>
          ) : (!bankAccounts || bankAccounts.length === 0) ? (
            <Text c="dimmed" size="sm">{t('bank_account.no_accounts')}</Text>
          ) : (
            <Table withRowBorders={false} verticalSpacing={4}>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th fz="xs">{t('bank_account.name_label')}</Table.Th>
                  <Table.Th fz="xs">{t('bank_account.account_number_label')}</Table.Th>
                  <Table.Th fz="xs">{t('bank_account.iban_label')}</Table.Th>
                  <Table.Th fz="xs">{t('bank_account.swift_label')}</Table.Th>
                  <Table.Th fz="xs">{t('bank_account.currency_label')}</Table.Th>
                  <Table.Th fz="xs">{t('bank_account.qr_type_label')}</Table.Th>
                  <Table.Th></Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {bankAccounts.map((ba) => (
                  <Table.Tr key={ba.id}>
                    <Table.Td fz="sm">
                      <Group gap="xs">
                        <Text size="sm">{ba.name}</Text>
                        {ba.is_default && <Badge size="xs" color="teal">{t('bank_account.default')}</Badge>}
                      </Group>
                    </Table.Td>
                    <Table.Td fz="sm">{ba.account_number}</Table.Td>
                    <Table.Td fz="sm">{ba.iban || '\u2014'}</Table.Td>
                    <Table.Td fz="sm">{ba.swift || '\u2014'}</Table.Td>
                    <Table.Td fz="sm">{ba.currency}</Table.Td>
                    <Table.Td fz="sm">{ba.qr_type || '\u2014'}</Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        <ActionIcon variant="subtle" size="sm" color="blue" onClick={() => onEdit(ba)}>
                          <IconPencil size={14} />
                        </ActionIcon>
                        <ActionIcon variant="subtle" size="sm" color="red" onClick={() => onDelete(ba)}>
                          <IconTrash size={14} />
                        </ActionIcon>
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}
        </Box>
        <Divider />
      </Table.Td>
    </Table.Tr>
  )
}

function MobileBankAccounts({ supplierId, onEdit, onDelete, onCreate }: {
  supplierId: string
  onEdit: (ba: BankAccount) => void
  onDelete: (ba: BankAccount) => void
  onCreate: (supplierId: string) => void
}) {
  const { t } = useT()
  const { data: bankAccounts, isLoading } = useQuery({
    queryKey: ['bank-accounts', supplierId],
    queryFn: () => api.getBankAccounts(supplierId),
  })

  if (isLoading) return <Center py="xs"><Loader size="xs" /></Center>

  return (
    <Stack gap="xs">
      <Group justify="space-between">
        <Group gap="xs">
          <IconBuildingBank size={14} />
          <Text size="xs" fw={600}>{t('bank_account.manage')}</Text>
        </Group>
        <Button size="compact-xs" variant="light" leftSection={<IconPlus size={12} />}
          onClick={() => onCreate(supplierId)}>
          {t('bank_account.add')}
        </Button>
      </Group>
      {(!bankAccounts || bankAccounts.length === 0) ? (
        <Text c="dimmed" size="xs">{t('bank_account.no_accounts')}</Text>
      ) : bankAccounts.map((ba) => (
        <Paper key={ba.id} p="xs" radius="sm" withBorder bg="var(--mantine-color-default-hover)">
          <Group justify="space-between" mb={2}>
            <Group gap="xs">
              <Text size="xs" fw={500}>{ba.name || ba.account_number}</Text>
              {ba.is_default && <Badge size="xs" color="teal">{t('bank_account.default')}</Badge>}
            </Group>
            <Group gap="xs">
              <ActionIcon variant="subtle" size="xs" color="blue" onClick={() => onEdit(ba)}>
                <IconPencil size={12} />
              </ActionIcon>
              <ActionIcon variant="subtle" size="xs" color="red" onClick={() => onDelete(ba)}>
                <IconTrash size={12} />
              </ActionIcon>
            </Group>
          </Group>
          {ba.name && <Text size="xs" c="dimmed">{ba.account_number}</Text>}
          {ba.iban && <Text size="xs" c="dimmed">IBAN: {ba.iban}</Text>}
          <Group gap="xs">
            {ba.swift && <Text size="xs" c="dimmed">SWIFT: {ba.swift}</Text>}
            <Text size="xs" c="dimmed">{ba.currency}</Text>
          </Group>
        </Paper>
      ))}
    </Stack>
  )
}

export function SupplierList() {
  const isMobile = useIsMobile()
  const [modalOpen, setModalOpen] = useState(false)
  const [editingSupplier, setEditingSupplier] = useState<Supplier | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Supplier | null>(null)

  // Bank account state — set of supplier IDs whose bank accounts are expanded
  const [expandedBanks, setExpandedBanks] = useState<Set<string>>(new Set())
  // SMTP config state — set of supplier IDs whose SMTP section is expanded
  const [expandedSmtp, setExpandedSmtp] = useState<Set<string>>(new Set())
  const [bankModalOpen, setBankModalOpen] = useState(false)
  const [editingBank, setEditingBank] = useState<BankAccount | null>(null)
  const [bankSupplierId, setBankSupplierId] = useState<string>('')
  const [bankDeleteOpen, setBankDeleteOpen] = useState(false)
  const [bankDeleteTarget, setBankDeleteTarget] = useState<BankAccount | null>(null)

  // Bank account form state
  const [baName, setBaName] = useState('')
  const [baAccountNumber, setBaAccountNumber] = useState('')
  const [baIban, setBaIban] = useState('')
  const [baSwift, setBaSwift] = useState('')
  const [baCurrency, setBaCurrency] = useState('CZK')
  const [baIsDefault, setBaIsDefault] = useState(false)
  const [baQrType, setBaQrType] = useState('spayd')

  // Supplier form state
  const [name, setName] = useState('')
  const [ico, setIco] = useState('')
  const [dic, setDic] = useState('')
  const [icDph, setIcDph] = useState('')
  const [street, setStreet] = useState('')
  const [city, setCity] = useState('')
  const [zip, setZip] = useState('')
  const [country, setCountry] = useState('CZ')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [website, setWebsite] = useState('')
  const [invoicePrefix, setInvoicePrefix] = useState('')
  const [isVatPayer, setIsVatPayer] = useState(false)
  const [isDefault, setIsDefault] = useState(false)
  const [notes, setNotes] = useState('')

  const queryClient = useQueryClient()
  const { t } = useT()

  const { data: suppliers, isLoading } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const supplierCount = (suppliers || []).length

  const uploadMutation = useMutation({
    mutationFn: ({ id, file }: { id: string; file: File }) => api.uploadLogo(id, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({ title: t('notify.logo_uploaded'), message: t('notify.logo_uploaded_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('notify.upload_failed'), message: err.message, color: 'red' })
    },
  })

  const deleteLogoMutation = useMutation({
    mutationFn: (id: string) => api.deleteLogo(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({ title: t('notify.logo_removed'), message: t('notify.logo_removed_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const saveMutation = useMutation({
    mutationFn: (data: Partial<Supplier>) =>
      editingSupplier ? api.updateSupplier(editingSupplier.id, data) : api.createSupplier(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({
        title: editingSupplier ? t('notify.supplier_updated') : t('notify.supplier_created'),
        message: t('notify.supplier_saved_msg').replace('{name}', name),
        color: 'green',
      })
      closeModal()
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteSupplier(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      notifications.show({ title: t('notify.supplier_deleted'), message: t('notify.supplier_deleted_msg'), color: 'green' })
      setDeleteOpen(false)
      setDeleteTarget(null)
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  // Bank account mutations
  const saveBankMutation = useMutation({
    mutationFn: (data: Partial<BankAccount>) =>
      editingBank
        ? api.updateBankAccount(editingBank.id, data)
        : api.createBankAccount(bankSupplierId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bank-accounts', bankSupplierId] })
      notifications.show({
        title: editingBank ? t('notify.bank_account_updated') : t('notify.bank_account_created'),
        message: editingBank ? t('notify.bank_account_updated_msg') : t('notify.bank_account_created_msg'),
        color: 'green',
      })
      closeBankModal()
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteBankMutation = useMutation({
    mutationFn: (id: string) => api.deleteBankAccount(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bank-accounts'] })
      notifications.show({ title: t('notify.bank_account_deleted'), message: t('notify.bank_account_deleted_msg'), color: 'green' })
      setBankDeleteOpen(false)
      setBankDeleteTarget(null)
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const openCreate = () => {
    setEditingSupplier(null)
    setName('')
    setIco('')
    setDic('')
    setIcDph('')
    setStreet('')
    setCity('')
    setZip('')
    setCountry('CZ')
    setEmail('')
    setPhone('')
    setWebsite('')
    setInvoicePrefix('')
    setIsVatPayer(false)
    setIsDefault(false)
    setNotes('')
    setModalOpen(true)
  }

  const openEdit = (supplier: Supplier) => {
    setEditingSupplier(supplier)
    setName(supplier.name)
    setIco(supplier.ico)
    setDic(supplier.dic)
    setIcDph(supplier.ic_dph)
    setStreet(supplier.street)
    setCity(supplier.city)
    setZip(supplier.zip)
    setCountry(supplier.country || 'CZ')
    setEmail(supplier.email)
    setPhone(supplier.phone)
    setWebsite(supplier.website)
    setInvoicePrefix(supplier.invoice_prefix)
    setIsVatPayer(supplier.is_vat_payer)
    setIsDefault(supplier.is_default)
    setNotes(supplier.notes)
    setModalOpen(true)
  }

  const closeModal = () => {
    setModalOpen(false)
    setEditingSupplier(null)
  }

  const openBankCreate = (supplierId: string) => {
    setEditingBank(null)
    setBankSupplierId(supplierId)
    setBaName('')
    setBaAccountNumber('')
    setBaIban('')
    setBaSwift('')
    setBaCurrency('CZK')
    setBaIsDefault(false)
    setBaQrType('spayd')
    setBankModalOpen(true)
  }

  const openBankEdit = (ba: BankAccount) => {
    setEditingBank(ba)
    setBankSupplierId(ba.supplier_id)
    setBaName(ba.name)
    setBaAccountNumber(ba.account_number)
    setBaIban(ba.iban)
    setBaSwift(ba.swift)
    setBaCurrency(ba.currency)
    setBaIsDefault(ba.is_default)
    setBaQrType(ba.qr_type || 'spayd')
    setBankModalOpen(true)
  }

  const closeBankModal = () => {
    setBankModalOpen(false)
    setEditingBank(null)
  }

  const handleSave = () => {
    if (!name.trim()) {
      notifications.show({ title: t('supplier.missing_name_title'), message: t('supplier.missing_name_msg'), color: 'orange' })
      return
    }
    saveMutation.mutate({
      name: name.trim(),
      ico: ico.trim(),
      dic: dic.trim(),
      ic_dph: icDph.trim(),
      street: street.trim(),
      city: city.trim(),
      zip: zip.trim(),
      country,
      email: email.trim(),
      phone: phone.trim(),
      website: website.trim(),
      invoice_prefix: invoicePrefix.trim(),
      is_vat_payer: isVatPayer,
      is_default: isDefault,
      notes: notes.trim(),
    })
  }

  const handleBankSave = () => {
    if (!baName.trim() && !baAccountNumber.trim()) {
      notifications.show({ title: t('bank_account.missing_fields_title'), message: t('bank_account.missing_fields_msg'), color: 'orange' })
      return
    }
    saveBankMutation.mutate({
      name: baName.trim(),
      account_number: baAccountNumber.trim(),
      iban: baIban.trim(),
      swift: baSwift.trim(),
      currency: baCurrency,
      is_default: baIsDefault,
      qr_type: baQrType,
    })
  }

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const pageTitle = supplierCount > 1 ? t('supplier.title_plural') : t('supplier.title')
  const pageSubtitle = supplierCount > 1 ? t('supplier.subtitle_plural') : t('supplier.subtitle')

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <div>
          <Title order={isMobile ? 3 : 2}>{pageTitle}</Title>
          <Text c="dimmed" size="sm">{pageSubtitle}</Text>
        </div>
        <Group gap="sm" wrap="wrap">
          {supplierCount > 0 && (
            <Button
              variant={expandedBanks.size > 0 ? 'filled' : 'light'}
              leftSection={<IconBuildingBank size={16} />}
              onClick={() => {
                if (expandedBanks.size > 0) {
                  setExpandedBanks(new Set())
                } else {
                  setExpandedBanks(new Set((suppliers || []).map(s => s.id)))
                }
              }}
            >
              {t('bank_account.manage')}
            </Button>
          )}
          {supplierCount > 0 && (
            <Button
              variant={expandedSmtp.size > 0 ? 'filled' : 'light'}
              leftSection={<IconAt size={16} />}
              onClick={() => {
                if (expandedSmtp.size > 0) {
                  setExpandedSmtp(new Set())
                } else {
                  setExpandedSmtp(new Set((suppliers || []).map(s => s.id)))
                }
              }}
            >
              {t('email.smtp_title')}
            </Button>
          )}
          <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>{t('supplier.add')}</Button>
        </Group>
      </Group>

      <Paper p="md" radius="md" withBorder>
        {supplierCount === 0 ? (
          <Text c="dimmed" size="sm" ta="center" py="xl">{t('supplier.no_suppliers')}</Text>
        ) : isMobile ? (
          <Stack gap="sm">
            {(suppliers || []).map((s) => (
              <Paper key={s.id} p="sm" radius="sm" withBorder>
                <Group mb="xs">
                  {s.logo_path ? (
                    <Avatar src={api.getLogoUrl(s.id)} size={40} radius="sm" />
                  ) : (
                    <Avatar size={40} radius="sm" color="gray">{s.name.charAt(0).toUpperCase()}</Avatar>
                  )}
                  <div style={{ flex: 1 }}>
                    <Group gap="xs">
                      <Text size="sm" fw={500}>{s.name}</Text>
                      {s.is_default && supplierCount > 1 && <Badge size="xs" color="blue">{t('supplier.default')}</Badge>}
                      <Badge size="xs" color={s.is_vat_payer ? 'green' : 'gray'} variant="light">
                        {s.is_vat_payer ? t('supplier.yes') : t('supplier.no')}
                      </Badge>
                    </Group>
                    <Text size="xs" c="dimmed">
                      {s.ico && `IČO: ${s.ico}`}{s.dic && ` | DIČ: ${s.dic}`}{s.ic_dph && ` | IČ DPH: ${s.ic_dph}`}
                    </Text>
                    <Text size="xs" c="dimmed">{[s.street, s.city, s.zip].filter(Boolean).join(', ')}</Text>
                  </div>
                  <Group gap="xs">
                    <FileButton onChange={(file) => { if (file) uploadMutation.mutate({ id: s.id, file }) }} accept="image/png,image/jpeg">
                      {(props) => (
                        <ActionIcon variant="light" size="sm" color="gray" {...props}><IconUpload size={14} /></ActionIcon>
                      )}
                    </FileButton>
                    <ActionIcon variant="light" size="sm" color="blue" onClick={() => openEdit(s)}>
                      <IconPencil size={14} />
                    </ActionIcon>
                    <ActionIcon variant="light" size="sm" color="red" onClick={() => { setDeleteTarget(s); setDeleteOpen(true) }}>
                      <IconTrash size={14} />
                    </ActionIcon>
                  </Group>
                </Group>
                <Divider my="xs" />
                <Box onClick={() => {
                  setExpandedBanks(prev => {
                    const next = new Set(prev)
                    if (next.has(s.id)) next.delete(s.id); else next.add(s.id)
                    return next
                  })
                }} style={{ cursor: 'pointer' }}>
                  <Group gap="xs">
                    <IconBuildingBank size={14} />
                    <Text size="xs" c="dimmed">{t('bank_account.manage')}</Text>
                  </Group>
                </Box>
                {expandedBanks.has(s.id) && (
                  <Box mt="xs">
                    <MobileBankAccounts
                      supplierId={s.id}
                      onEdit={openBankEdit}
                      onDelete={(ba) => { setBankDeleteTarget(ba); setBankDeleteOpen(true) }}
                      onCreate={openBankCreate}
                    />
                  </Box>
                )}
                <Box onClick={() => {
                  setExpandedSmtp(prev => {
                    const next = new Set(prev)
                    if (next.has(s.id)) next.delete(s.id); else next.add(s.id)
                    return next
                  })
                }} style={{ cursor: 'pointer' }}>
                  <Group gap="xs">
                    <IconAt size={14} />
                    <Text size="xs" c="dimmed">{t('email.smtp_title')}</Text>
                  </Group>
                </Box>
                {expandedSmtp.has(s.id) && (
                  <Box mt="xs" p="xs" bg="var(--mantine-color-default-hover)" style={{ borderRadius: 'var(--mantine-radius-sm)' }}>
                    <SmtpConfigForm supplierId={s.id} supplierName={s.name} supplierEmail={s.email} />
                  </Box>
                )}
              </Paper>
            ))}
          </Stack>
        ) : (
          <Box style={{ overflowX: 'auto' }}>
          <Table style={{ minWidth: 700 }}>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>{t('supplier.logo')}</Table.Th>
                <Table.Th>{t('supplier.name')}</Table.Th>
                <Table.Th>{t('supplier.ico')}</Table.Th>
                <Table.Th>{t('supplier.dic')}</Table.Th>
                <Table.Th>{t('supplier.address')}</Table.Th>
                <Table.Th>{t('supplier.vat_payer')}</Table.Th>
                <Table.Th></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {(suppliers || []).map((s) => (
                <>
                <Table.Tr key={s.id} style={{ cursor: 'pointer' }} onClick={() => {
                  setExpandedBanks(prev => {
                    const next = new Set(prev)
                    if (next.has(s.id)) next.delete(s.id); else next.add(s.id)
                    return next
                  })
                }}>
                  <Table.Td onClick={(e) => e.stopPropagation()}>
                    <Group gap="xs">
                      {s.logo_path ? (
                        <Avatar
                          src={api.getLogoUrl(s.id)}
                          size={36}
                          radius="sm"
                        />
                      ) : (
                        <Avatar size={36} radius="sm" color="gray">
                          {s.name.charAt(0).toUpperCase()}
                        </Avatar>
                      )}
                      <Group gap={4}>
                        <FileButton
                          onChange={(file) => {
                            if (file) uploadMutation.mutate({ id: s.id, file })
                          }}
                          accept="image/png,image/jpeg"
                        >
                          {(props) => (
                            <ActionIcon variant="subtle" size="sm" color="blue" {...props}>
                              <IconUpload size={14} />
                            </ActionIcon>
                          )}
                        </FileButton>
                        {s.logo_path && (
                          <ActionIcon
                            variant="subtle"
                            size="sm"
                            color="red"
                            onClick={() => deleteLogoMutation.mutate(s.id)}
                          >
                            <IconTrash size={14} />
                          </ActionIcon>
                        )}
                      </Group>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <Text size="sm" fw={500}>{s.name}</Text>
                      {s.is_default && supplierCount > 1 && <Badge size="xs" color="blue">{t('supplier.default')}</Badge>}
                    </Group>
                  </Table.Td>
                  <Table.Td fz="sm">{s.ico}</Table.Td>
                  <Table.Td fz="sm">{s.dic || '\u2014'}{s.ic_dph && ` | IČ DPH: ${s.ic_dph}`}</Table.Td>
                  <Table.Td fz="sm">{[s.street, s.city, s.zip].filter(Boolean).join(', ')}</Table.Td>
                  <Table.Td>
                    <Badge size="xs" color={s.is_vat_payer ? 'green' : 'gray'} variant="light">
                      {s.is_vat_payer ? t('supplier.yes') : t('supplier.no')}
                    </Badge>
                  </Table.Td>
                  <Table.Td onClick={(e) => e.stopPropagation()}>
                    <Group gap="xs">
                      <ActionIcon variant="light" size="sm" color="gray" onClick={() => {
                        setExpandedSmtp(prev => {
                          const next = new Set(prev)
                          if (next.has(s.id)) next.delete(s.id); else next.add(s.id)
                          return next
                        })
                      }}>
                        <IconAt size={14} />
                      </ActionIcon>
                      <ActionIcon variant="light" size="sm" color="blue" onClick={() => openEdit(s)}>
                        <IconPencil size={14} />
                      </ActionIcon>
                      <ActionIcon variant="light" size="sm" color="red" onClick={() => { setDeleteTarget(s); setDeleteOpen(true) }}>
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
                {expandedBanks.has(s.id) && (
                  <BankAccountsRow
                    key={`${s.id}-bank`}
                    supplierId={s.id}
                    supplierName={s.name}
                    onEdit={openBankEdit}
                    onDelete={(ba) => { setBankDeleteTarget(ba); setBankDeleteOpen(true) }}
                    onCreate={openBankCreate}
                  />
                )}
                {expandedSmtp.has(s.id) && (
                  <Table.Tr key={`${s.id}-smtp`}>
                    <Table.Td colSpan={7} p={0}>
                      <Box px="lg" py="md" bg="var(--mantine-color-default-hover)">
                        <SmtpConfigForm supplierId={s.id} supplierName={s.name} supplierEmail={s.email} />
                      </Box>
                      <Divider />
                    </Table.Td>
                  </Table.Tr>
                )}
                </>
              ))}
            </Table.Tbody>
          </Table>
          </Box>
        )}
      </Paper>

      {/* Supplier create/edit modal */}
      <Modal opened={modalOpen} onClose={closeModal}
        title={editingSupplier ? t('supplier.edit_title') : t('supplier.new_title')} size="lg" fullScreen={isMobile}>
        <Stack gap="md">
          <TextInput label={t('supplier.name_label')} value={name}
            onChange={(e) => setName(e.currentTarget.value)} required />
          <SimpleGrid cols={{ base: 1, sm: 3 }}>
            <TextInput label={t('supplier.ico_label')} value={ico}
              onChange={(e) => setIco(e.currentTarget.value)} />
            <TextInput label={t('supplier.dic_label')} value={dic}
              onChange={(e) => setDic(e.currentTarget.value)} />
            <div>
              <Text size="sm" fw={500} mb={4}>{t('supplier.is_vat_payer_label')}</Text>
              <SegmentedControl
                value={isVatPayer ? 'yes' : 'no'}
                onChange={(v) => setIsVatPayer(v === 'yes')}
                data={[
                  { label: t('supplier.no'), value: 'no' },
                  { label: t('supplier.yes'), value: 'yes' },
                ]}
                fullWidth
              />
            </div>
          </SimpleGrid>
          {country.toUpperCase() === 'SK' && (
            <TextInput label={t('supplier.ic_dph_label')} value={icDph}
              onChange={(e) => setIcDph(e.currentTarget.value)} />
          )}
          <TextInput label={t('supplier.street_label')} value={street}
            onChange={(e) => setStreet(e.currentTarget.value)} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('supplier.city_label')} value={city}
              onChange={(e) => setCity(e.currentTarget.value)} />
            <TextInput label={t('supplier.zip_label')} value={zip}
              onChange={(e) => setZip(e.currentTarget.value)} />
          </SimpleGrid>
          <CountrySelect label={t('supplier.country_label')}
            value={country} onChange={(v) => setCountry(v || 'CZ')} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('supplier.email_label')} value={email}
              onChange={(e) => setEmail(e.currentTarget.value)} />
            <TextInput label={t('supplier.phone_label')} value={phone}
              onChange={(e) => setPhone(e.currentTarget.value)} />
          </SimpleGrid>
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('supplier.website_label')} value={website}
              onChange={(e) => setWebsite(e.currentTarget.value)} />
            <TextInput label={
              <Group gap={4}>
                <span>{t('supplier.invoice_prefix_label')}</span>
                <Tooltip label={t('supplier.invoice_prefix_hint')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
                  <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
                </Tooltip>
              </Group>
            } value={invoicePrefix}
              onChange={(e) => setInvoicePrefix(e.currentTarget.value)} />
          </SimpleGrid>
          <Textarea label={t('supplier.notes_label')} value={notes}
            onChange={(e) => setNotes(e.currentTarget.value)} minRows={2} />
          {supplierCount > 1 && editingSupplier && (
            <Switch label={t('supplier.default')}
              checked={isDefault}
              onChange={(e) => setIsDefault(e.currentTarget.checked)}
              disabled={editingSupplier.is_default} />
          )}
          <Group justify="end" mt="md">
            <Button variant="default" onClick={closeModal}>{t('common.cancel')}</Button>
            <Button onClick={handleSave} loading={saveMutation.isPending}>
              {editingSupplier ? t('common.save') : t('common.create')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Supplier delete modal */}
      <Modal opened={deleteOpen} onClose={() => { setDeleteOpen(false); setDeleteTarget(null) }}
        title={t('supplier.delete_title')} size="sm">
        <Stack gap="md">
          <Text size="sm" dangerouslySetInnerHTML={{
            __html: t('supplier.delete_confirm').replace('{name}', deleteTarget?.name || '')
          }} />
          <Group justify="end">
            <Button variant="default" onClick={() => { setDeleteOpen(false); setDeleteTarget(null) }}>{t('common.cancel')}</Button>
            <Button color="red" onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              loading={deleteMutation.isPending}>{t('common.delete')}</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Bank account create/edit modal */}
      <Modal opened={bankModalOpen} onClose={closeBankModal}
        title={editingBank ? t('bank_account.edit_title') : t('bank_account.new_title')} size="md" fullScreen={isMobile}>
        <Stack gap="md">
          <TextInput label={t('bank_account.name_label')} value={baName}
            onChange={(e) => setBaName(e.currentTarget.value)} />
          <TextInput label={t('bank_account.account_number_label')} value={baAccountNumber}
            onChange={(e) => setBaAccountNumber(e.currentTarget.value)} required />
          <TextInput label={t('bank_account.iban_label')} value={baIban}
            onChange={(e) => setBaIban(e.currentTarget.value)} />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <TextInput label={t('bank_account.swift_label')} value={baSwift}
              onChange={(e) => setBaSwift(e.currentTarget.value)} />
            <Select label={t('bank_account.currency_label')}
              data={['CZK', 'EUR', 'USD', 'GBP', 'PLN']}
              value={baCurrency} onChange={(v) => setBaCurrency(v || 'CZK')}
              allowDeselect={false} />
          </SimpleGrid>
          <Select label={t('bank_account.qr_type_label')}
            data={[
              { value: 'spayd', label: t('bank_account.qr_spayd') },
              { value: 'pay_by_square', label: t('bank_account.qr_pbs') },
              { value: 'epc', label: t('bank_account.qr_epc') },
              { value: 'none', label: t('bank_account.qr_none') },
            ]}
            value={baQrType} onChange={(v) => setBaQrType(v || 'spayd')}
            allowDeselect={false} />
          <Switch label={t('bank_account.is_default_label')} checked={baIsDefault}
            onChange={(e) => setBaIsDefault(e.currentTarget.checked)}
            disabled={editingBank?.is_default} />
          <Group justify="end" mt="md">
            <Button variant="default" onClick={closeBankModal}>{t('common.cancel')}</Button>
            <Button onClick={handleBankSave} loading={saveBankMutation.isPending}>
              {editingBank ? t('common.save') : t('common.create')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Bank account delete modal */}
      <Modal opened={bankDeleteOpen} onClose={() => { setBankDeleteOpen(false); setBankDeleteTarget(null) }}
        title={t('bank_account.delete_title')} size="sm">
        <Stack gap="md">
          <Text size="sm" dangerouslySetInnerHTML={{
            __html: t('bank_account.delete_confirm').replace('{name}', bankDeleteTarget?.name || bankDeleteTarget?.account_number || '')
          }} />
          <Group justify="end">
            <Button variant="default" onClick={() => { setBankDeleteOpen(false); setBankDeleteTarget(null) }}>{t('common.cancel')}</Button>
            <Button color="red" onClick={() => bankDeleteTarget && deleteBankMutation.mutate(bankDeleteTarget.id)}
              loading={deleteBankMutation.isPending}>{t('common.delete')}</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
