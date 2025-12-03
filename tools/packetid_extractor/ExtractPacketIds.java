// 工具：從 named client jar 抽取各協議的封包 ID 對應表，輸出 JSON。
import java.lang.reflect.Field;
import java.util.*;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import net.minecraft.SharedConstants;
import net.minecraft.network.ProtocolInfo;
import net.minecraft.network.protocol.PacketType;
import net.minecraft.network.protocol.configuration.ConfigurationProtocols;
import net.minecraft.network.protocol.game.GameProtocols;
import net.minecraft.network.protocol.login.LoginProtocols;
import net.minecraft.server.Bootstrap;

public class ExtractPacketIds {
  // PacketType 實例到常數名稱的映射，用於輸出易讀名稱。
  private static final IdentityHashMap<PacketType<?>, String> TYPE_NAME = new IdentityHashMap<>();
  // PacketType 來源類別名稱，方便追蹤來自哪個 *PacketTypes。
  private static final IdentityHashMap<PacketType<?>, String> TYPE_SOURCE = new IdentityHashMap<>();

  /**
   * 掃描指定類別中的公開 PacketType 常數並收集名稱。
   * @param cls 含有封包常數的 *PacketTypes 類別
   */
  private static void collectTypeConstants(Class<?> cls) throws Exception {
    for (Field f : cls.getFields()) {
      if (PacketType.class.isAssignableFrom(f.getType())) {
        PacketType<?> pt = (PacketType<?>) f.get(null);
        TYPE_NAME.put(pt, f.getName());
        TYPE_SOURCE.put(pt, cls.getSimpleName());
      }
    }
  }

  /**
   * 透過 ProtocolInfo.DetailsProvider 取得封包列表，並轉為可序列化的 map。
   * @param provider 提供封包列表的協議物件（SimpleUnboundProtocol 或 UnboundProtocol）
   * @return 封包資料列表，包含 id / flow / packet_type / name / source
   */
  private static List<Map<String, Object>> extract(ProtocolInfo.DetailsProvider provider) {
    List<Map<String, Object>> list = new ArrayList<>();
    provider.details().listPackets((type, id) -> {
      Map<String, Object> m = new LinkedHashMap<>();
      m.put("id", id);
      m.put("flow", type.flow().toString());
      m.put("packet_type", type.id().toString());
      m.put("name", TYPE_NAME.getOrDefault(type, type.id().toString()));
      m.put("source", TYPE_SOURCE.getOrDefault(type, ""));
      list.add(m);
    });
    return list;
  }

  /**
   * 入口：初始化遊戲版本、Bootstrap，收集三種主要協議（game/configuration/login）的封包 ID，並輸出 JSON。
   */
  public static void main(String[] args) throws Exception {
    // 使用簡易 log4j 設定避免寫檔。
    if (System.getProperty("log4j.configurationFile") == null) {
      System.setProperty("log4j.configurationFile", "/tmp/log4j2.xml");
    }
    SharedConstants.tryDetectVersion();
    Bootstrap.bootStrap();
    // 還原 stdout，避免 wrapStreams 改寫。
    System.setOut(Bootstrap.STDOUT);

    collectTypeConstants(net.minecraft.network.protocol.game.GamePacketTypes.class);
    collectTypeConstants(net.minecraft.network.protocol.common.CommonPacketTypes.class);
    collectTypeConstants(net.minecraft.network.protocol.cookie.CookiePacketTypes.class);
    collectTypeConstants(net.minecraft.network.protocol.configuration.ConfigurationPacketTypes.class);
    collectTypeConstants(net.minecraft.network.protocol.login.LoginPacketTypes.class);
    collectTypeConstants(net.minecraft.network.protocol.handshake.HandshakePacketTypes.class);
    collectTypeConstants(net.minecraft.network.protocol.status.StatusPacketTypes.class);
    collectTypeConstants(net.minecraft.network.protocol.ping.PingPacketTypes.class);

    Map<String, Object> root = new LinkedHashMap<>();
    root.put("protocol_version", SharedConstants.getProtocolVersion());

    Map<String, Object> protocols = new LinkedHashMap<>();
    Map<String, Object> game = new LinkedHashMap<>();
    game.put("clientbound", extract(GameProtocols.CLIENTBOUND_TEMPLATE));
    game.put("serverbound", extract(GameProtocols.SERVERBOUND_TEMPLATE));
    protocols.put("game", game);

    Map<String, Object> configuration = new LinkedHashMap<>();
    configuration.put("clientbound", extract(ConfigurationProtocols.CLIENTBOUND_TEMPLATE));
    configuration.put("serverbound", extract(ConfigurationProtocols.SERVERBOUND_TEMPLATE));
    protocols.put("configuration", configuration);

    Map<String, Object> login = new LinkedHashMap<>();
    login.put("clientbound", extract(LoginProtocols.CLIENTBOUND_TEMPLATE));
    login.put("serverbound", extract(LoginProtocols.SERVERBOUND_TEMPLATE));
    protocols.put("login", login);

    root.put("protocols", protocols);

    Gson gson = new GsonBuilder().setPrettyPrinting().create();
    System.out.println(gson.toJson(root));
  }
}
