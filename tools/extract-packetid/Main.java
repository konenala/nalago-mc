import java.io.*;
import java.lang.reflect.*;
import java.net.*;
import java.nio.file.*;
import java.util.*;

public class Main {
    public static void main(String[] args) throws Exception {
        if (args.length < 3) {
            System.err.println("Usage: java Main <deobf-jar> <state:play|login|configuration|status> <output-json>");
            return;
        }
        String jarPath = args[0];
        String state = args[1].toUpperCase();
        String out = args[2];

        URLClassLoader cl = new URLClassLoader(new URL[]{Paths.get(jarPath).toUri().toURL()});
        Class<?> connectionProtocol = cl.loadClass("net.minecraft.network.ConnectionProtocol");
        Object[] protocols = connectionProtocol.getEnumConstants();

        Map<String,List<Map<String,Object>>> result = new LinkedHashMap<>();
        for (Object proto : protocols) {
            if (!proto.toString().equals(state)) continue;
            Object protoInfo = findProtocolInfo(proto);
            Map<String,List<Map<String,Object>>> dirMap = extract(protoInfo);
            result.putAll(dirMap);
        }
        writeJson(out, result);
    }

    private static Object findProtocolInfo(Object proto) throws Exception {
        for (Method m : proto.getClass().getDeclaredMethods()) {
            if (m.getParameterCount()==0 && m.getReturnType().getSimpleName().toLowerCase().contains("protocolinfo")) {
                m.setAccessible(true);
                return m.invoke(proto);
            }
        }
        throw new RuntimeException("ProtocolInfo getter not found");
    }

    private static Map<String,List<Map<String,Object>>> extract(Object protoInfo) throws Exception {
        Map<String,List<Map<String,Object>>> dirMap = new LinkedHashMap<>();
        for (Method m : protoInfo.getClass().getDeclaredMethods()) {
            if (m.getParameterCount()==0 && Map.class.isAssignableFrom(m.getReturnType())) {
                m.setAccessible(true);
                @SuppressWarnings("unchecked") Map<Object,Object> mp = (Map<Object,Object>) m.invoke(protoInfo);
                for (Map.Entry<Object,Object> e : mp.entrySet()) {
                    String flow = e.getKey().toString().toLowerCase().contains("clientbound")?"client":
                            e.getKey().toString().toLowerCase().contains("serverbound")?"server":e.getKey().toString();
                    Object idMapObj = e.getValue();
                    List<Map<String,Object>> lst = dirMap.computeIfAbsent(flow, k-> new ArrayList<>());
                    Method entrySet = idMapObj.getClass().getMethod("entrySet");
                    for (Object entry : (Iterable<?>) entrySet.invoke(idMapObj)) {
                        Map.Entry<?,?> en = (Map.Entry<?,?>) entry;
                        int id = ((Number) en.getKey()).intValue();
                        Object pktType = en.getValue();
                        lst.add(Map.of("id", id, "name", pktType.toString()));
                    }
                }
            }
        }
        return dirMap;
    }

    private static void writeJson(String out, Map<String,List<Map<String,Object>>> result) throws Exception {
        StringBuilder sb = new StringBuilder();
        sb.append("{\n");
        int di=0; for (var d: result.entrySet()) {
            sb.append("  \"").append(d.getKey()).append("\": [\n");
            List<Map<String,Object>> list = d.getValue();
            list.sort(Comparator.comparingInt(m->(Integer)m.get("id")));
            for (int i=0;i<list.size();i++) {
                var m=list.get(i);
                sb.append("    {\"id\": ").append(m.get("id")).append(", \"name\": \"").append(m.get("name")).append("\"}");
                if (i+1<list.size()) sb.append(",");
                sb.append("\n");
            }
            sb.append("  ]"); if (++di<result.size()) sb.append(","); sb.append("\n");
        }
        sb.append("}\n");
        Files.writeString(Paths.get(out), sb.toString());
    }
}
