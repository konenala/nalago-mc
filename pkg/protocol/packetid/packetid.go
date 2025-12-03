package packetid

// 本地臨時 packetid 常數，依當前代碼引用自動列出；數值以宣告順序遞增。
type ClientboundPacketID int32
type ServerboundPacketID int32

const (
	ClientboundAbilities                   ClientboundPacketID = iota
	ClientboundAcknowledgePlayerDigging    ClientboundPacketID = iota
	ClientboundActionBar                   ClientboundPacketID = iota
	ClientboundAdvancements                ClientboundPacketID = iota
	ClientboundAnimation                   ClientboundPacketID = iota
	ClientboundAttachEntity                ClientboundPacketID = iota
	ClientboundBlockAction                 ClientboundPacketID = iota
	ClientboundBlockBreakAnimation         ClientboundPacketID = iota
	ClientboundBlockChange                 ClientboundPacketID = iota
	ClientboundBossBar                     ClientboundPacketID = iota
	ClientboundCamera                      ClientboundPacketID = iota
	ClientboundChatSuggestions             ClientboundPacketID = iota
	ClientboundChunkBatchFinished          ClientboundPacketID = iota
	ClientboundChunkBatchStart             ClientboundPacketID = iota
	ClientboundChunkBiomes                 ClientboundPacketID = iota
	ClientboundClearTitles                 ClientboundPacketID = iota
	ClientboundCloseWindow                 ClientboundPacketID = iota
	ClientboundCollect                     ClientboundPacketID = iota
	ClientboundConfigClearDialog           ClientboundPacketID = iota
	ClientboundConfigCookieRequest         ClientboundPacketID = iota
	ClientboundConfigCustomPayload         ClientboundPacketID = iota
	ClientboundConfigCustomReportDetails   ClientboundPacketID = iota
	ClientboundConfigDisconnect            ClientboundPacketID = iota
	ClientboundConfigFinishConfiguration   ClientboundPacketID = iota
	ClientboundConfigKeepAlive             ClientboundPacketID = iota
	ClientboundConfigPing                  ClientboundPacketID = iota
	ClientboundConfigRegistryData          ClientboundPacketID = iota
	ClientboundConfigResetChat             ClientboundPacketID = iota
	ClientboundConfigResourcePackPop       ClientboundPacketID = iota
	ClientboundConfigResourcePackPush      ClientboundPacketID = iota
	ClientboundConfigSelectKnownPacks      ClientboundPacketID = iota
	ClientboundConfigServerLinks           ClientboundPacketID = iota
	ClientboundConfigShowDialog            ClientboundPacketID = iota
	ClientboundConfigStoreCookie           ClientboundPacketID = iota
	ClientboundConfigTransfer              ClientboundPacketID = iota
	ClientboundConfigUpdateEnabledFeatures ClientboundPacketID = iota
	ClientboundConfigUpdateTags            ClientboundPacketID = iota
	ClientboundCraftProgressBar            ClientboundPacketID = iota
	ClientboundCraftRecipeResponse         ClientboundPacketID = iota
	ClientboundCustomPayload               ClientboundPacketID = iota
	ClientboundDamageEvent                 ClientboundPacketID = iota
	ClientboundDeathCombatEvent            ClientboundPacketID = iota
	ClientboundDebugSample                 ClientboundPacketID = iota
	ClientboundDeclareCommands             ClientboundPacketID = iota
	ClientboundDeclareRecipes              ClientboundPacketID = iota
	ClientboundDifficulty                  ClientboundPacketID = iota
	ClientboundDisconnect                  ClientboundPacketID = iota
	ClientboundEndCombatEvent              ClientboundPacketID = iota
	ClientboundEnterCombatEvent            ClientboundPacketID = iota
	ClientboundEntityDestroy               ClientboundPacketID = iota
	ClientboundEntityEffect                ClientboundPacketID = iota
	ClientboundEntityEquipment             ClientboundPacketID = iota
	ClientboundEntityHeadRotation          ClientboundPacketID = iota
	ClientboundEntityLook                  ClientboundPacketID = iota
	ClientboundEntityMetadata              ClientboundPacketID = iota
	ClientboundEntityMoveLook              ClientboundPacketID = iota
	ClientboundEntitySoundEffect           ClientboundPacketID = iota
	ClientboundEntityStatus                ClientboundPacketID = iota
	ClientboundEntityTeleport              ClientboundPacketID = iota
	ClientboundEntityUpdateAttributes      ClientboundPacketID = iota
	ClientboundEntityVelocity              ClientboundPacketID = iota
	ClientboundExperience                  ClientboundPacketID = iota
	ClientboundExplosion                   ClientboundPacketID = iota
	ClientboundFacePlayer                  ClientboundPacketID = iota
	ClientboundGameStateChange             ClientboundPacketID = iota
	ClientboundHeldItemSlot                ClientboundPacketID = iota
	ClientboundHideMessage                 ClientboundPacketID = iota
	ClientboundHurtAnimation               ClientboundPacketID = iota
	ClientboundInitializeWorldBorder       ClientboundPacketID = iota
	ClientboundKeepAlive                   ClientboundPacketID = iota
	ClientboundKickDisconnect              ClientboundPacketID = iota
	ClientboundLogin                       ClientboundPacketID = iota
	ClientboundLoginCookieRequest          ClientboundPacketID = iota
	ClientboundLoginCustomQuery            ClientboundPacketID = iota
	ClientboundLoginHello                  ClientboundPacketID = iota
	ClientboundLoginLoginCompression       ClientboundPacketID = iota
	ClientboundLoginLoginDisconnect        ClientboundPacketID = iota
	ClientboundLoginLoginFinished          ClientboundPacketID = iota
	ClientboundMap                         ClientboundPacketID = iota
	ClientboundMapChunk                    ClientboundPacketID = iota
	ClientboundMoveMinecart                ClientboundPacketID = iota
	ClientboundMultiBlockChange            ClientboundPacketID = iota
	ClientboundNbtQueryResponse            ClientboundPacketID = iota
	ClientboundOpenBook                    ClientboundPacketID = iota
	ClientboundOpenHorseWindow             ClientboundPacketID = iota
	ClientboundOpenSignEntity              ClientboundPacketID = iota
	ClientboundOpenWindow                  ClientboundPacketID = iota
	ClientboundPing                        ClientboundPacketID = iota
	ClientboundPingResponse                ClientboundPacketID = iota
	ClientboundPlayerChat                  ClientboundPacketID = iota
	ClientboundPlayerInfo                  ClientboundPacketID = iota
	ClientboundPlayerRemove                ClientboundPacketID = iota
	ClientboundPlayerRotation              ClientboundPacketID = iota
	ClientboundPlayerlistHeader            ClientboundPacketID = iota
	ClientboundPosition                    ClientboundPacketID = iota
	ClientboundProfilelessChat             ClientboundPacketID = iota
	ClientboundRecipeBookAdd               ClientboundPacketID = iota
	ClientboundRecipeBookRemove            ClientboundPacketID = iota
	ClientboundRecipeBookSettings          ClientboundPacketID = iota
	ClientboundRelEntityMove               ClientboundPacketID = iota
	ClientboundRemoveEntityEffect          ClientboundPacketID = iota
	ClientboundResetScore                  ClientboundPacketID = iota
	ClientboundRespawn                     ClientboundPacketID = iota
	ClientboundScoreboardDisplayObjective  ClientboundPacketID = iota
	ClientboundScoreboardObjective         ClientboundPacketID = iota
	ClientboundScoreboardScore             ClientboundPacketID = iota
	ClientboundSelectAdvancementTab        ClientboundPacketID = iota
	ClientboundServerData                  ClientboundPacketID = iota
	ClientboundSetCooldown                 ClientboundPacketID = iota
	ClientboundSetCursorItem               ClientboundPacketID = iota
	ClientboundSetPassengers               ClientboundPacketID = iota
	ClientboundSetPlayerInventory          ClientboundPacketID = iota
	ClientboundSetProjectilePower          ClientboundPacketID = iota
	ClientboundSetSlot                     ClientboundPacketID = iota
	ClientboundSetTickingState             ClientboundPacketID = iota
	ClientboundSetTitleSubtitle            ClientboundPacketID = iota
	ClientboundSetTitleText                ClientboundPacketID = iota
	ClientboundSetTitleTime                ClientboundPacketID = iota
	ClientboundShowDialog                  ClientboundPacketID = iota
	ClientboundSimulationDistance          ClientboundPacketID = iota
	ClientboundSoundEffect                 ClientboundPacketID = iota
	ClientboundSpawnEntity                 ClientboundPacketID = iota
	ClientboundSpawnPosition               ClientboundPacketID = iota
	ClientboundStartConfiguration          ClientboundPacketID = iota
	ClientboundStatistics                  ClientboundPacketID = iota
	ClientboundStepTick                    ClientboundPacketID = iota
	ClientboundStopSound                   ClientboundPacketID = iota
	ClientboundSyncEntityPosition          ClientboundPacketID = iota
	ClientboundSystemChat                  ClientboundPacketID = iota
	ClientboundTabComplete                 ClientboundPacketID = iota
	ClientboundTags                        ClientboundPacketID = iota
	ClientboundTeams                       ClientboundPacketID = iota
	ClientboundTestInstanceBlockStatus     ClientboundPacketID = iota
	ClientboundTileEntityData              ClientboundPacketID = iota
	ClientboundTrackedWaypoint             ClientboundPacketID = iota
	ClientboundTradeList                   ClientboundPacketID = iota
	ClientboundUnloadChunk                 ClientboundPacketID = iota
	ClientboundUpdateHealth                ClientboundPacketID = iota
	ClientboundUpdateLight                 ClientboundPacketID = iota
	ClientboundUpdateTime                  ClientboundPacketID = iota
	ClientboundUpdateViewDistance          ClientboundPacketID = iota
	ClientboundUpdateViewPosition          ClientboundPacketID = iota
	ClientboundVehicleMove                 ClientboundPacketID = iota
	ClientboundWindowItems                 ClientboundPacketID = iota
	ClientboundWorldBorderCenter           ClientboundPacketID = iota
	ClientboundWorldBorderLerpSize         ClientboundPacketID = iota
	ClientboundWorldBorderSize             ClientboundPacketID = iota
	ClientboundWorldBorderWarningDelay     ClientboundPacketID = iota
	ClientboundWorldBorderWarningReach     ClientboundPacketID = iota
	ClientboundWorldEvent                  ClientboundPacketID = iota
	ClientboundWorldParticles              ClientboundPacketID = iota
)

const (
	ServerboundAbilities ServerboundPacketID = iota
	ServerboundAcceptTeleportation
	ServerboundAdvancementTab
	ServerboundArmAnimation
	ServerboundBlockDig
	ServerboundBlockPlace
	ServerboundChangeGamemode
	ServerboundChat
	ServerboundChatAck
	ServerboundChatCommand
	ServerboundChatCommandSigned
	ServerboundChatMessage
	ServerboundChatSessionUpdate
	ServerboundChunkBatchReceived
	ServerboundClientCommand
	ServerboundCloseWindow
	ServerboundConfigClientInformation
	ServerboundConfigCookieResponse
	ServerboundConfigCustomClickAction
	ServerboundConfigCustomPayload
	ServerboundConfigFinishConfiguration
	ServerboundConfigKeepAlive
	ServerboundConfigPong
	ServerboundConfigResourcePack
	ServerboundConfigSelectKnownPacks
	ServerboundConfigurationAcknowledged
	ServerboundContainerClick
	ServerboundContainerClose
	ServerboundCraftRecipeRequest
	ServerboundCustomPayload
	ServerboundDebugSampleSubscription
	ServerboundDisplayedRecipe
	ServerboundEditBook
	ServerboundEnchantItem
	ServerboundEntityAction
	ServerboundFlying
	ServerboundGenerateStructure
	ServerboundHeldItemSlot
	ServerboundKeepAlive
	ServerboundLockDifficulty
	ServerboundLoginCookieResponse
	ServerboundLoginCustomQueryAnswer
	ServerboundLoginHello
	ServerboundLoginKey
	ServerboundLoginLoginAcknowledged
	ServerboundLook
	ServerboundMessageAcknowledgement
	ServerboundMovePlayerPos
	ServerboundMovePlayerPosRot
	ServerboundMovePlayerRot
	ServerboundNameItem
	ServerboundPickItemFromBlock
	ServerboundPickItemFromEntity
	ServerboundPingRequest
	ServerboundPlayerAction
	ServerboundPlayerInput
	ServerboundPlayerLoaded
	ServerboundPong
	ServerboundPosition
	ServerboundPositionLook
	ServerboundQueryBlockNbt
	ServerboundQueryEntityNbt
	ServerboundRecipeBook
	ServerboundResourcePackReceive
	ServerboundSelectBundleItem
	ServerboundSelectTrade
	ServerboundSetBeaconEffect
	ServerboundSetCreativeSlot
	ServerboundSetDifficulty
	ServerboundSetSlotState
	ServerboundSetTestBlock
	ServerboundSpectate
	ServerboundSteerBoat
	ServerboundTabComplete
	ServerboundTeleportConfirm
	ServerboundTestInstanceBlockAction
	ServerboundTickEnd
	ServerboundUpdateCommandBlock
	ServerboundUpdateCommandBlockMinecart
	ServerboundUpdateJigsawBlock
	ServerboundUpdateSign
	ServerboundUpdateStructureBlock
	ServerboundUseEntity
	ServerboundUseItem
	ServerboundUseItemOn
	ServerboundVehicleMove
	ServerboundWindowClick
)
