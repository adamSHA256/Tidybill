import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Table,
  Badge,
  Button,
  TextInput,
  NumberInput,
  Select,
  Modal,
  ActionIcon,
  Loader,
  Center,
  Box,
  SimpleGrid,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconSearch, IconPlus, IconPencil, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, formatMoney, type Item } from '../api/client'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'

const ADD_VAT_RATE = '__add_vat_rate__'
const ADD_UNIT = '__add_unit__'

export function ItemCatalog() {
  const isMobile = useIsMobile()
  const [search, setSearch] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<Item | null>(null)

  // Form state
  const [description, setDescription] = useState('')
  const [defaultPrice, setDefaultPrice] = useState<number>(0)
  const [defaultUnit, setDefaultUnit] = useState('ks')
  const [defaultVATRate, setDefaultVATRate] = useState<number>(21)
  const [category, setCategory] = useState('')
  const [vatRateModalOpen, setVatRateModalOpen] = useState(false)
  const [newVatRateValue, setNewVatRateValue] = useState('')
  const [unitModalOpen, setUnitModalOpen] = useState(false)
  const [newUnitValue, setNewUnitValue] = useState('')

  const queryClient = useQueryClient()
  const { t } = useT()

  const { data: items, isLoading } = useQuery({
    queryKey: ['items', search],
    queryFn: () => api.getItems(search || undefined),
  })

  const { data: categories } = useQuery({
    queryKey: ['item-categories'],
    queryFn: api.getItemCategories,
  })

  const { data: units } = useQuery({
    queryKey: ['units'],
    queryFn: api.getUnits,
  })
  const unitOptions = (units || []).map((u) => u.name)

  const { data: vatRates } = useQuery({
    queryKey: ['vat-rates'],
    queryFn: api.getVATRates,
  })
  const vatRateOptions = (vatRates || []).map((r) => String(r.rate))

  const createMutation = useMutation({
    mutationFn: (data: Partial<Item>) =>
      editingItem ? api.updateItem(editingItem.id, data) : api.createItem(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['items'] })
      queryClient.invalidateQueries({ queryKey: ['item-categories'] })
      notifications.show({
        title: editingItem ? t('notify.item_updated') : t('notify.item_created'),
        message: t('notify.item_saved_msg').replace('{description}', description),
        color: 'green',
      })
      closeModal()
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: api.deleteItem,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['items'] })
      notifications.show({ title: t('notify.item_deleted'), message: t('notify.item_deleted_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const openCreate = () => {
    setEditingItem(null)
    setDescription('')
    setDefaultPrice(0)
    setDefaultUnit((units || []).find((u) => u.is_default)?.name || unitOptions[0] || 'ks')
    setDefaultVATRate((vatRates || []).find((r) => r.is_default)?.rate ?? 21)
    setCategory('')
    setModalOpen(true)
  }

  const openEdit = (item: Item) => {
    setEditingItem(item)
    setDescription(item.description)
    setDefaultPrice(item.default_price)
    setDefaultUnit(item.default_unit)
    setDefaultVATRate(item.default_vat_rate)
    setCategory(item.category)
    setModalOpen(true)
  }

  const closeModal = () => {
    setModalOpen(false)
    setEditingItem(null)
  }

  const handleSave = () => {
    if (!description.trim()) {
      notifications.show({ title: t('item.missing_desc_title'), message: t('item.missing_desc_msg'), color: 'orange' })
      return
    }
    createMutation.mutate({
      description: description.trim(),
      default_price: defaultPrice,
      default_unit: defaultUnit,
      default_vat_rate: defaultVATRate,
      category: category.trim(),
    })
  }

  const handleAddVatRate = () => {
    const rate = parseFloat(newVatRateValue.trim())
    if (isNaN(rate) || rate < 0) return
    const currentRates = vatRates || []
    if (!currentRates.some((r) => r.rate === rate)) {
      api.updateVATRates([...currentRates, { rate }]).then(() => {
        queryClient.invalidateQueries({ queryKey: ['vat-rates'] })
      })
    }
    setDefaultVATRate(rate)
    setVatRateModalOpen(false)
  }

  const handleAddUnit = () => {
    const name = newUnitValue.trim()
    if (!name) return
    const currentUnits = units || []
    if (!currentUnits.some((u) => u.name === name)) {
      api.updateUnits([...currentUnits, { name }]).then(() => {
        queryClient.invalidateQueries({ queryKey: ['units'] })
      })
    }
    setDefaultUnit(name)
    setUnitModalOpen(false)
  }

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const categoryOptions = (categories || []).map((c) => ({ value: c, label: c }))

  return (
    <Stack gap="lg">
      <Group justify="space-between" wrap="wrap">
        <div>
          <Title order={isMobile ? 3 : 2}>{t('item.title')}</Title>
          <Text c="dimmed" size="sm">{t('item.subtitle')}</Text>
        </div>
        <Group wrap="wrap">
          <TextInput
            placeholder={t('item.search')}
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
          />
          <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>{t('item.add')}</Button>
        </Group>
      </Group>

      <Paper p="md" radius="md" withBorder>
        {(items || []).length === 0 ? (
          <Text c="dimmed" size="sm" ta="center" py="xl">
            {search ? t('item.no_match') : t('item.no_items')}
          </Text>
        ) : isMobile ? (
          <Stack gap="sm">
            {(items || []).map((item) => (
              <Paper key={item.id} p="sm" radius="sm" withBorder>
                <Group justify="space-between" mb={4}>
                  <Text size="sm" fw={500}>{item.description}</Text>
                  <Group gap="xs">
                    <ActionIcon variant="light" size="sm" color="blue" onClick={() => openEdit(item)}>
                      <IconPencil size={14} />
                    </ActionIcon>
                    <ActionIcon variant="light" size="sm" color="red"
                      onClick={() => deleteMutation.mutate(item.id)}>
                      <IconTrash size={14} />
                    </ActionIcon>
                  </Group>
                </Group>
                <Group gap="xs">
                  <Text size="xs" c="dimmed">{formatMoney(item.default_price)} / {item.default_unit}</Text>
                  <Text size="xs" c="dimmed">{item.default_vat_rate}%</Text>
                  {item.category && <Badge size="xs" variant="light">{item.category}</Badge>}
                  <Badge size="xs" variant="light" color={item.usage_count > 0 ? 'blue' : 'gray'}>{item.usage_count}x</Badge>
                </Group>
              </Paper>
            ))}
          </Stack>
        ) : (
          <Box style={{ overflowX: 'auto' }}>
          <Table style={{ minWidth: 600 }}>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>{t('item.description')}</Table.Th>
                <Table.Th>{t('item.category')}</Table.Th>
                <Table.Th>{t('item.price')}</Table.Th>
                <Table.Th>{t('item.unit')}</Table.Th>
                <Table.Th>{t('item.vat')}</Table.Th>
                <Table.Th>{t('item.used')}</Table.Th>
                <Table.Th></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {(items || []).map((item) => (
                <Table.Tr key={item.id}>
                  <Table.Td>
                    <Text size="sm" fw={500}>{item.description}</Text>
                  </Table.Td>
                  <Table.Td>
                    {item.category ? (
                      <Badge size="xs" variant="light">{item.category}</Badge>
                    ) : (
                      <Text size="sm" c="dimmed">—</Text>
                    )}
                  </Table.Td>
                  <Table.Td fz="sm">{formatMoney(item.default_price)}</Table.Td>
                  <Table.Td fz="sm">{item.default_unit}</Table.Td>
                  <Table.Td fz="sm">{item.default_vat_rate}%</Table.Td>
                  <Table.Td>
                    <Badge size="xs" variant="light" color={item.usage_count > 0 ? 'blue' : 'gray'}>
                      {item.usage_count}x
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <ActionIcon variant="light" size="sm" color="blue" onClick={() => openEdit(item)}>
                        <IconPencil size={14} />
                      </ActionIcon>
                      <ActionIcon variant="light" size="sm" color="red"
                        onClick={() => deleteMutation.mutate(item.id)}>
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
          </Box>
        )}

        {(items || []).length > 0 && (
          <Group justify="space-between" mt="md">
            <Text size="sm" c="dimmed">{t('item.showing').replace('{count}', String((items || []).length))}</Text>
          </Group>
        )}
      </Paper>

      <Modal opened={modalOpen} onClose={closeModal}
        title={editingItem ? t('item.edit_title') : t('item.new_title')} size="md" fullScreen={isMobile}>
        <Stack gap="md">
          <TextInput label={t('item.description_label')} placeholder={t('item.description_placeholder')}
            value={description} onChange={(e) => setDescription(e.currentTarget.value)} required />
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <NumberInput label={t('item.default_price')} min={0} value={defaultPrice}
              onChange={(val) => setDefaultPrice(Number(val) || 0)} />
            <Select label={t('item.unit_label')}
              data={[
                ...(unitOptions.length > 0 ? unitOptions : ['ks', 'hod', 'den', 'm\u00B2']).map((u) => {
                  const key = 'unit.' + u; const translated = t(key); return { value: u, label: translated !== key ? translated : u }
                }),
                { value: ADD_UNIT, label: `+ ${t('invoice.add_unit')}` },
              ]}
              value={defaultUnit}
              onChange={(v) => {
                if (v === ADD_UNIT) { setNewUnitValue(''); setUnitModalOpen(true); return }
                setDefaultUnit(v || unitOptions[0] || 'ks')
              }} />
          </SimpleGrid>
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <Select label={t('item.vat_rate')}
              data={[
                ...(vatRateOptions.length > 0 ? vatRateOptions : ['0', '12', '21']).map((r) => ({ value: r, label: `${r}%` })),
                { value: ADD_VAT_RATE, label: `+ ${t('invoice.add_vat_rate')}` },
              ]}
              value={String(defaultVATRate)}
              onChange={(v) => {
                if (v === ADD_VAT_RATE) { setNewVatRateValue(''); setVatRateModalOpen(true); return }
                setDefaultVATRate(Number(v) || 0)
              }} />
            <TextInput label={t('item.category_label')} placeholder={t('item.category_placeholder')}
              value={category} onChange={(e) => setCategory(e.currentTarget.value)}
              description={categoryOptions.length > 0 ? t('item.existing_categories').replace('{categories}', (categories || []).join(', ')) : undefined} />
          </SimpleGrid>
          <Group justify="end" mt="md">
            <Button variant="default" onClick={closeModal}>{t('common.cancel')}</Button>
            <Button onClick={handleSave} loading={createMutation.isPending}>
              {editingItem ? t('common.save') : t('common.create')}
            </Button>
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
            <Button disabled={!newVatRateValue.trim()} onClick={handleAddVatRate}>{t('common.save')}</Button>
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
            <Button disabled={!newUnitValue.trim()} onClick={handleAddUnit}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
